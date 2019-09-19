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

package matching

import (
	"errors"
	"fmt"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/utils"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"math"
	"time"
)

const (
	orderIdWindowCap = 10000
)

type orderBook struct {
	// 每一个product都会对应一个order book
	product *models.Product

	// 深度，asks & bids
	depths map[models.Side]*depth

	// 严格连续递增的交易id，用于在trade的主键id
	tradeSeq int64

	// 严格连续递增的日志seq，用于写入撮合日志
	logSeq int64

	// 防止order被重复提交到orderBook中，采用滑动窗口去重策略
	orderIdWindow Window
}

// orderBook快照，定时保存快照用于快速启动恢复
type orderBookSnapshot struct {
	// 对应的product id
	ProductId string

	// orderBook中的全量订单
	Orders []BookOrder

	// 当前tradeSeq
	TradeSeq int64

	// 当前logSeq
	LogSeq int64

	// 去重窗口
	OrderIdWindow Window
}

type depth struct {
	// 保存所有正在book上的order
	orders map[int64]*BookOrder

	// 价格优先的priceLevel队列，用于获取level2
	// Price -> *PriceLevel
	levels *treemap.Map

	// 价格优先，时间优先的订单队列，用于订单match
	// priceOrderIdKey -> orderId
	queue *treemap.Map
}

type PriceLevel struct {
	Price      decimal.Decimal
	Size       decimal.Decimal
	OrderCount int64
}

type priceOrderIdKey struct {
	price   decimal.Decimal
	orderId int64
}

type BookOrder struct {
	OrderId int64
	Size    decimal.Decimal
	Funds   decimal.Decimal
	Price   decimal.Decimal
	Side    models.Side
	Type    models.OrderType
}

func NewOrderBook(product *models.Product) *orderBook {
	asks := &depth{
		levels: treemap.NewWith(utils.DecimalAscComparator),
		queue:  treemap.NewWith(priceOrderIdKeyAscComparator),
		orders: map[int64]*BookOrder{},
	}
	bids := &depth{
		levels: treemap.NewWith(utils.DecimalDescComparator),
		queue:  treemap.NewWith(priceOrderIdKeyDescComparator),
		orders: map[int64]*BookOrder{},
	}

	orderBook := &orderBook{
		product:       product,
		depths:        map[models.Side]*depth{models.SideBuy: bids, models.SideSell: asks},
		orderIdWindow: newWindow(0, orderIdWindowCap),
	}
	return orderBook
}

func (o *orderBook) ApplyOrder(order *models.Order) (logs []Log) {
	// 订单去重，防止订单被重复提交到撮合引擎
	err := o.orderIdWindow.put(order.Id)
	if err != nil {
		log.Error(err)
		return logs
	}

	takerOrder := &BookOrder{
		OrderId: order.Id,
		Size:    order.Size,
		Funds:   order.Funds,
		Price:   order.Price,
		Side:    order.Side,
		Type:    order.Type,
	}

	// 如果是market-buy订单，将price设置成无限制高，如果是market-sell，将price设置成0，这样可以确保价格一定会交叉
	if takerOrder.Type == models.OrderTypeMarket {
		if takerOrder.Side == models.SideBuy {
			takerOrder.Price = decimal.NewFromFloat(math.MaxFloat32)
		} else {
			takerOrder.Price = decimal.Zero
		}
	}

	makerDepth := o.depths[takerOrder.Side.Opposite()]
	for itr := makerDepth.queue.Iterator(); itr.Next(); {
		orderId := itr.Value().(int64)
		makerOrder := makerDepth.orders[orderId]

		// 判断taker和maker是否发生价格交叉
		if (takerOrder.Side == models.SideBuy && takerOrder.Price.Cmp(makerOrder.Price) < 0) ||
			(takerOrder.Side == models.SideSell && takerOrder.Price.Cmp(makerOrder.Price) > 0) {
			break
		}

		var size decimal.Decimal
		var price decimal.Decimal
		if takerOrder.Type == models.OrderTypeLimit ||
			(takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideSell) {
			if takerOrder.Size.IsZero() {
				break
			}

			// 成交价格
			price = makerOrder.Price

			// 取taker和maker的最小size做为成交size
			size = decimal.Min(takerOrder.Size, makerOrder.Size)

			// taker和maker都需要减掉size
			takerOrder.Size = takerOrder.Size.Sub(size)
			err := makerDepth.decrSize(makerOrder.OrderId, size)
			if err != nil {
				log.Fatal(err)
			}
		} else if takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideBuy {
			if takerOrder.Funds.IsZero() {
				break
			}

			// 成交价格
			price = makerOrder.Price

			// 计算以当前价格taker的size
			takerSize := takerOrder.Funds.Div(price).Truncate(o.product.BaseScale)
			if takerSize.IsZero() {
				break
			}

			// 取taker和maker的最小size做为成交size
			size = decimal.Min(takerSize, makerOrder.Size)
			funds := size.Mul(price)

			// taker减去funds，maker减去size
			takerOrder.Funds = takerOrder.Funds.Sub(funds)
			err := makerDepth.decrSize(makerOrder.OrderId, size)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("unknown orderType and side combination")
		}

		// match成功，递增一次tradeId，用于这一次交易的序列号
		matchLog := &MatchLog{
			Base:         Base{LogTypeMatch, o.nextLogSeq(), o.product.Id, time.Now()},
			TradeId:      o.nextTradeSeq(),
			TakerOrderId: takerOrder.OrderId,
			MakerOrderId: makerOrder.OrderId,
			Side:         makerOrder.Side,
			Price:        price,
			Size:         size,
		}
		logs = append(logs, matchLog)

		// maker被完全fill
		if makerOrder.Size.IsZero() {
			doneLog := &DoneLog{
				Base:          Base{LogTypeDone, o.nextLogSeq(), o.product.Id, time.Now()},
				OrderId:       makerOrder.OrderId,
				Price:         makerOrder.Price,
				RemainingSize: makerOrder.Size,
				Reason:        models.DoneReasonFilled,
				Side:          makerOrder.Side,
			}
			logs = append(logs, doneLog)
		}
	}

	if takerOrder.Type == models.OrderTypeLimit && takerOrder.Size.GreaterThan(decimal.Zero) {
		// limit taker还有未成交的size，则把taker放入orderBook
		o.depths[takerOrder.Side].add(*takerOrder)

		openLog := &OpenLog{
			Base:          Base{LogTypeOpen, o.nextLogSeq(), o.product.Id, time.Now()},
			OrderId:       takerOrder.OrderId,
			RemainingSize: takerOrder.Size,
			Price:         takerOrder.Price,
			Side:          takerOrder.Side,
		}
		logs = append(logs, openLog)

	} else {
		var price = takerOrder.Price
		var remainingSize = takerOrder.Size
		var reason = models.DoneReasonFilled

		if takerOrder.Type == models.OrderTypeMarket {
			price = decimal.Zero
			remainingSize = decimal.Zero
			if takerOrder.Side == models.SideSell {
				if takerOrder.Size.GreaterThan(decimal.Zero) {
					reason = models.DoneReasonCancelled
				}
			} else {
				if takerOrder.Funds.GreaterThan(decimal.Zero) {
					reason = models.DoneReasonCancelled
				}
			}
		}

		doneLog := &DoneLog{
			Base:          Base{LogTypeDone, o.nextLogSeq(), o.product.Id, time.Now()},
			OrderId:       takerOrder.OrderId,
			Price:         price,
			RemainingSize: remainingSize,
			Reason:        reason,
			Side:          takerOrder.Side,
		}
		logs = append(logs, doneLog)
	}
	return logs
}

func (o *orderBook) CancelOrder(order *models.Order) (logs []Log) {
	_ = o.orderIdWindow.put(order.Id)

	bookOrder, found := o.depths[order.Side].orders[order.Id]
	if !found {
		return logs
	}

	// 将order的size全部decr，等于remove操作
	remainingSize := bookOrder.Size
	err := o.depths[order.Side].decrSize(order.Id, bookOrder.Size)
	if err != nil {
		panic(err)
	}

	doneLog := &DoneLog{
		Base:          Base{LogTypeDone, o.nextLogSeq(), o.product.Id, time.Now()},
		OrderId:       bookOrder.OrderId,
		Price:         bookOrder.Price,
		RemainingSize: remainingSize,
		Reason:        models.DoneReasonCancelled,
		Side:          bookOrder.Side,
	}
	return append(logs, doneLog)
}

func (o *orderBook) Snapshot() orderBookSnapshot {
	snapshot := orderBookSnapshot{
		Orders:        []BookOrder{},
		LogSeq:        o.logSeq,
		TradeSeq:      o.tradeSeq,
		OrderIdWindow: o.orderIdWindow,
	}

	for _, order := range o.depths[models.SideSell].orders {
		snapshot.Orders = append(snapshot.Orders, *order)
	}
	for _, order := range o.depths[models.SideBuy].orders {
		snapshot.Orders = append(snapshot.Orders, *order)
	}

	return snapshot
}

func (o *orderBook) Restore(snapshot *orderBookSnapshot) {
	o.logSeq = snapshot.LogSeq
	o.tradeSeq = snapshot.TradeSeq
	o.orderIdWindow = snapshot.OrderIdWindow
	if o.orderIdWindow.Cap == 0 {
		o.orderIdWindow = newWindow(0, orderIdWindowCap)
	}

	for _, order := range snapshot.Orders {
		o.depths[order.Side].add(order)
	}
}

func (o *orderBook) nextLogSeq() int64 {
	o.logSeq++
	return o.logSeq
}

func (o *orderBook) nextTradeSeq() int64 {
	o.tradeSeq++
	return o.tradeSeq
}

func (d *depth) add(order BookOrder) {
	d.orders[order.OrderId] = &order

	d.queue.Put(&priceOrderIdKey{order.Price, order.OrderId}, order.OrderId)

	val, found := d.levels.Get(order.Price)
	if !found {
		d.levels.Put(order.Price, &PriceLevel{
			Price:      order.Price,
			Size:       order.Size,
			OrderCount: 1,
		})
	} else {
		level := val.(*PriceLevel)
		level.Size = level.Size.Add(order.Size)
		level.OrderCount++
	}
}

func (d *depth) decrSize(orderId int64, size decimal.Decimal) error {
	order, found := d.orders[orderId]
	if !found {
		return errors.New(fmt.Sprintf("order %v not found on book", orderId))
	}

	if order.Size.LessThan(size) {
		return errors.New(fmt.Sprintf("order %v Size %v less than %v", orderId, order.Size, size))
	}

	var removed bool
	order.Size = order.Size.Sub(size)
	if order.Size.IsZero() {
		delete(d.orders, orderId)
		removed = true
	}

	// 订单被移除出orderBook，清理priceTime队列
	if removed {
		d.queue.Remove(&priceOrderIdKey{order.Price, order.OrderId})
	}

	val, _ := d.levels.Get(order.Price)
	level := val.(*PriceLevel)
	level.Size = level.Size.Sub(size)
	if level.Size.IsZero() {
		d.levels.Remove(order.Price)
	} else if removed {
		level.OrderCount--
	}

	return nil
}

func priceOrderIdKeyAscComparator(a, b interface{}) int {
	aAsserted := a.(*priceOrderIdKey)
	bAsserted := b.(*priceOrderIdKey)

	x := aAsserted.price.Cmp(bAsserted.price)
	if x != 0 {
		return x
	}

	y := aAsserted.orderId - bAsserted.orderId
	if y == 0 {
		return 0
	} else if y > 0 {
		return 1
	} else {
		return -1
	}
}

func priceOrderIdKeyDescComparator(a, b interface{}) int {
	aAsserted := a.(*priceOrderIdKey)
	bAsserted := b.(*priceOrderIdKey)

	x := aAsserted.price.Cmp(bAsserted.price)
	if x != 0 {
		return -x
	}
	decimal.Zero.Neg()

	y := aAsserted.orderId - bAsserted.orderId
	if y == 0 {
		return 0
	} else if y > 0 {
		return 1
	} else {
		return -1
	}
}
