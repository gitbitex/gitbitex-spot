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

	var changedLevel *matching.PriceLevel

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
			changedLevel = &matching.PriceLevel{
				Price:      price,
				Size:       newSize,
				OrderCount: 1,
			}
			priceLevels.Put(price, changedLevel)
		} else {
			changedLevel = val.(*matching.PriceLevel)
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

		changedLevel = val.(*matching.PriceLevel)
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

func (s *orderBook) SnapshotLevel2() OrderBookLevel2Snapshot {
	snapshot := OrderBookLevel2Snapshot{
		ProductId: s.productId,
		Seq:       s.seq,
		Asks:      [][3]interface{}{},
		Bids:      [][3]interface{}{},
	}
	for itr := s.depths[models.SideBuy].Iterator(); itr.Next(); {
		v := itr.Value().(*matching.PriceLevel)
		snapshot.Bids = append(snapshot.Bids, [3]interface{}{v.Price.String(), v.Size.String(), v.OrderCount})
	}
	for itr := s.depths[models.SideSell].Iterator(); itr.Next(); {
		v := itr.Value().(*matching.PriceLevel)
		snapshot.Asks = append(snapshot.Asks, [3]interface{}{v.Price.String(), v.Size.String(), v.OrderCount})
	}
	return snapshot
}

func (s *orderBook) SnapshotFull() OrderBookFullSnapshot {
	snapshot := OrderBookFullSnapshot{
		ProductId: s.productId,
		Seq:       s.seq,
		LogOffset: s.logOffset,
		LogSeq:    s.logSeq,
		Orders:    []matching.BookOrder{},
	}
	for _, order := range s.orders {
		snapshot.Orders = append(snapshot.Orders, *order)
	}
	return snapshot
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

type snapshotStore struct {
	redisClient *redis.Client
}

var store *snapshotStore
var onceStore sync.Once

func sharedSnapshotStore() *snapshotStore {
	onceStore.Do(func() {
		gbeConfig, err := conf.GetConfig()
		if err != nil {
			panic(err)
		}

		redisClient := redis.NewClient(&redis.Options{
			Addr:     gbeConfig.Redis.Addr,
			Password: gbeConfig.Redis.Password,
			DB:       0,
		})

		store = &snapshotStore{redisClient: redisClient}
	})
	return store
}

func (s *snapshotStore) storeLevel2(productId string, snapshot *OrderBookLevel2Snapshot) error {
	buf, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	ret := s.redisClient.Set(orderBookL2SnapshotKeyPrefix+productId, string(buf), 7*24*time.Hour)
	return ret.Err()
}

func (s *snapshotStore) storeFull(productId string, snapshot *OrderBookFullSnapshot) error {
	buf, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	ret := s.redisClient.Set(orderBookFullSnapshotKeyPrefix+productId, string(buf), 7*24*time.Hour)
	return ret.Err()
}

func (s *snapshotStore) getLastLevel2(productId string) (*OrderBookLevel2Snapshot, error) {
	ret, err := s.redisClient.Get(orderBookL2SnapshotKeyPrefix + productId).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		} else {
			return nil, err
		}
	}

	var snapshot OrderBookLevel2Snapshot
	err = json.Unmarshal([]byte(ret), &snapshot)
	if err != nil {
		return nil, err
	}
	return &snapshot, err
}

func (s *snapshotStore) getLastFull(productId string) (*OrderBookFullSnapshot, error) {
	ret, err := s.redisClient.Get(orderBookFullSnapshotKeyPrefix + productId).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		} else {
			return nil, err
		}
	}

	var snapshot OrderBookFullSnapshot
	err = json.Unmarshal([]byte(ret), &snapshot)
	if err != nil {
		return nil, err
	}
	return &snapshot, err
}
