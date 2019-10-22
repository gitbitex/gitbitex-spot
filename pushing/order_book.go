// Copyright 2019 GitBitEx.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pushing

import (
	"encoding/json"
	"fmt"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gitbitex/gitbitex-spot/conf"
	"github.com/gitbitex/gitbitex-spot/matching"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/utils"
	"github.com/go-redis/redis"
	"github.com/shopspring/decimal"
	"sync"
	"time"
)

const (
	orderBookL2SnapshotKeyPrefix   = "order_book_level2_snapshot_"
	orderBookFullSnapshotKeyPrefix = "order_book_full_snapshot_"
)

type orderBook struct {
	productId string
	seq       int64
	logOffset int64
	logSeq    int64
	depths    map[models.Side]*treemap.Map
	orders    map[int64]*matching.BookOrder
}

type OrderBookLevel2Snapshot struct {
	ProductId string
	Seq       int64
	Asks      [][3]interface{}
	Bids      [][3]interface{}
}

type OrderBookFullSnapshot struct {
	ProductId string
	Seq       int64
	LogOffset int64
	LogSeq    int64
	Orders    []matching.BookOrder
}

type PriceLevel struct {
	Price      decimal.Decimal
	Size       decimal.Decimal
	OrderCount int64
}

func newOrderBook(productId string) *orderBook {
	b := &orderBook{
		productId: productId,
		depths:    map[models.Side]*treemap.Map{},
		orders:    map[int64]*matching.BookOrder{},
	}
	b.depths[models.SideBuy] = treemap.NewWith(utils.DecimalDescComparator)
	b.depths[models.SideSell] = treemap.NewWith(utils.DecimalAscComparator)
	return b
}

func (s *orderBook) saveOrder(logOffset, logSeq int64, orderId int64, newSize, price decimal.Decimal,
	side models.Side) *Level2Change {
	if newSize.LessThan(decimal.Zero) {
		panic(newSize)
	}

	var changedLevel *PriceLevel

	priceLevels := s.depths[side]
	order, found := s.orders[orderId]
	if !found {
		if newSize.IsZero() {
			return nil
		}

		s.orders[orderId] = &matching.BookOrder{
			OrderId: orderId,
			Size:    newSize,
			Side:    side,
			Price:   price,
		}

		val, found := priceLevels.Get(price)
		if !found {
			changedLevel = &PriceLevel{
				Price:      price,
				Size:       newSize,
				OrderCount: 1,
			}
			priceLevels.Put(price, changedLevel)
		} else {
			changedLevel = val.(*PriceLevel)
			changedLevel.Size = changedLevel.Size.Add(newSize)
			changedLevel.OrderCount++
		}

	} else {
		oldSize := order.Size
		decrSize := oldSize.Sub(newSize)
		order.Size = newSize

		var removed bool
		if order.Size.IsZero() {
			delete(s.orders, order.OrderId)
			removed = true
		}

		val, found := priceLevels.Get(price)
		if !found {
			panic(fmt.Sprintf("%v %v %v %v", orderId, price, newSize, side))
		}

		changedLevel = val.(*PriceLevel)
		changedLevel.Size = changedLevel.Size.Sub(decrSize)
		if changedLevel.Size.IsZero() {
			priceLevels.Remove(price)
		} else if removed {
			changedLevel.OrderCount--
		}
	}

	s.logOffset = logOffset
	s.logSeq = logSeq
	s.seq++
	return &Level2Change{
		ProductId: s.productId,
		Seq:       s.seq,
		Side:      side.String(),
		Price:     changedLevel.Price.String(),
		Size:      changedLevel.Size.String(),
	}
}

func (s *orderBook) SnapshotLevel2(levels int) *OrderBookLevel2Snapshot {
	snapshot := OrderBookLevel2Snapshot{
		ProductId: s.productId,
		Seq:       s.seq,
		Asks:      make([][3]interface{}, utils.MinInt(levels, s.depths[models.SideSell].Size())),
		Bids:      make([][3]interface{}, utils.MinInt(levels, s.depths[models.SideBuy].Size())),
	}
	for itr, i := s.depths[models.SideBuy].Iterator(), 0; itr.Next() && i < levels; i++ {
		v := itr.Value().(*PriceLevel)
		snapshot.Bids[i] = [3]interface{}{v.Price.String(), v.Size.String(), v.OrderCount}
	}
	for itr, i := s.depths[models.SideSell].Iterator(), 0; itr.Next() && i < levels; i++ {
		v := itr.Value().(*PriceLevel)
		snapshot.Asks[i] = [3]interface{}{v.Price.String(), v.Size.String(), v.OrderCount}
	}
	return &snapshot
}

func (s *orderBook) SnapshotFull() *OrderBookFullSnapshot {
	snapshot := OrderBookFullSnapshot{
		ProductId: s.productId,
		Seq:       s.seq,
		LogOffset: s.logOffset,
		LogSeq:    s.logSeq,
		Orders:    make([]matching.BookOrder, len(s.orders)),
	}

	i := 0
	for _, order := range s.orders {
		snapshot.Orders[i] = *order
		i++
	}
	return &snapshot
}

func (s *orderBook) Restore(snapshot *OrderBookFullSnapshot) {
	for _, order := range snapshot.Orders {
		s.saveOrder(0, 0, order.OrderId, order.Size, order.Price, order.Side)
	}
	s.productId = snapshot.ProductId
	s.seq = snapshot.Seq
	s.logOffset = snapshot.LogOffset
	s.logSeq = snapshot.LogSeq
}

// redisSnapshotStore is used to manage snapshots
type redisSnapshotStore struct {
	redisClient *redis.Client
}

var store *redisSnapshotStore
var onceStore sync.Once

func sharedSnapshotStore() *redisSnapshotStore {
	onceStore.Do(func() {
		gbeConfig := conf.GetConfig()

		redisClient := redis.NewClient(&redis.Options{
			Addr:     gbeConfig.Redis.Addr,
			Password: gbeConfig.Redis.Password,
			DB:       0,
		})

		store = &redisSnapshotStore{redisClient: redisClient}
	})
	return store
}

func (s *redisSnapshotStore) storeLevel2(productId string, snapshot *OrderBookLevel2Snapshot) error {
	buf, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return s.redisClient.Set(orderBookL2SnapshotKeyPrefix+productId, buf, 7*24*time.Hour).Err()
}

func (s *redisSnapshotStore) getLastLevel2(productId string) (*OrderBookLevel2Snapshot, error) {
	ret, err := s.redisClient.Get(orderBookL2SnapshotKeyPrefix + productId).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		} else {
			return nil, err
		}
	}

	var snapshot OrderBookLevel2Snapshot
	err = json.Unmarshal(ret, &snapshot)
	return &snapshot, err
}

func (s *redisSnapshotStore) storeFull(productId string, snapshot *OrderBookFullSnapshot) error {
	buf, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return s.redisClient.Set(orderBookFullSnapshotKeyPrefix+productId, buf, 7*24*time.Hour).Err()
}

func (s *redisSnapshotStore) getLastFull(productId string) (*OrderBookFullSnapshot, error) {
	ret, err := s.redisClient.Get(orderBookFullSnapshotKeyPrefix + productId).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		} else {
			return nil, err
		}
	}

	var snapshot OrderBookFullSnapshot
	err = json.Unmarshal(ret, &snapshot)
	return &snapshot, err
}
