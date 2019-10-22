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
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"math"
)

const (
	orderIdWindowCap = 10000
)

type orderBook struct {
	// one product corresponds to one order book
	product *models.Product

	// depths: asks & bids
	depths map[models.Side]*depth

	// strictly continuously increasing transaction ID, used for the primary key ID of trade
	tradeSeq int64

	// strictly continuously increasing log SEQ, used to write matching log
	logSeq int64

	// to prevent the order from being submitted to the order book repeatedly,
	// a sliding window de duplication strategy is adopted.
	orderIdWindow Window
}

type orderBookSnapshot struct {
	// order book product id
	ProductId string

	// all orders
	Orders []BookOrder

	// trade seq at snapshot time
	TradeSeq int64

	// log seq at snapshot time
	LogSeq int64

	// state of de duplication window
	OrderIdWindow Window
}

type priceOrderIdKey struct {
	price   decimal.Decimal
	orderId int64
}

func NewOrderBook(product *models.Product) *orderBook {
	asks := &depth{
		queue:  treemap.NewWith(priceOrderIdKeyAscComparator),
		orders: map[int64]*BookOrder{},
	}
	bids := &depth{
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
	// prevent orders from being submitted repeatedly to the matching engine
	err := o.orderIdWindow.put(order.Id)
	if err != nil {
		log.Error(err)
		return logs
	}

	takerOrder := newBookOrder(order)

	// If it's a Market-Buy order, set price to infinite high, and if it's market-sell,
	// set price to zero, which ensures that prices will cross.
	if takerOrder.Type == models.OrderTypeMarket {
		if takerOrder.Side == models.SideBuy {
			takerOrder.Price = decimal.NewFromFloat(math.MaxFloat32)
		} else {
			takerOrder.Price = decimal.Zero
		}
	}

	makerDepth := o.depths[takerOrder.Side.Opposite()]
	for itr := makerDepth.queue.Iterator(); itr.Next(); {
		makerOrder := makerDepth.orders[itr.Value().(int64)]

		// check whether there is price crossing between the taker and the maker
		if (takerOrder.Side == models.SideBuy && takerOrder.Price.LessThan(makerOrder.Price)) ||
			(takerOrder.Side == models.SideSell && takerOrder.Price.GreaterThan(makerOrder.Price)) {
			break
		}

		// trade price
		var price = makerOrder.Price
		// trade size
		var size decimal.Decimal

		if takerOrder.Type == models.OrderTypeLimit ||
			(takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideSell) {
			if takerOrder.Size.IsZero() {
				break
			}

			// Take the minimum size of taker and maker as trade size
			size = decimal.Min(takerOrder.Size, makerOrder.Size)

			// adjust the size of taker order
			takerOrder.Size = takerOrder.Size.Sub(size)

		} else if takerOrder.Type == models.OrderTypeMarket && takerOrder.Side == models.SideBuy {
			if takerOrder.Funds.IsZero() {
				break
			}

			// calculate the size of taker at current price
			takerSize := takerOrder.Funds.Div(price).Truncate(o.product.BaseScale)
			if takerSize.IsZero() {
				break
			}

			// Take the minimum size of taker and maker as trade size
			size = decimal.Min(takerSize, makerOrder.Size)
			funds := size.Mul(price)

			// adjust the funds of taker order
			takerOrder.Funds = takerOrder.Funds.Sub(funds)
		} else {
			log.Fatal("unknown orderType and side combination")
		}

		// adjust the size of maker order
		err := makerDepth.decrSize(makerOrder.OrderId, size)
		if err != nil {
			log.Fatal(err)
		}

		// matched,write a log
		matchLog := newMatchLog(o.nextLogSeq(), o.product.Id, o.nextTradeSeq(), takerOrder, makerOrder, price, size)
		logs = append(logs, matchLog)

		// maker is filled
		if makerOrder.Size.IsZero() {
			doneLog := newDoneLog(o.nextLogSeq(), o.product.Id, makerOrder, makerOrder.Size, models.DoneReasonFilled)
			logs = append(logs, doneLog)
		}
	}

	if takerOrder.Type == models.OrderTypeLimit && takerOrder.Size.GreaterThan(decimal.Zero) {
		// If taker has an uncompleted size, put taker in orderBook
		o.depths[takerOrder.Side].add(*takerOrder)

		openLog := newOpenLog(o.nextLogSeq(), o.product.Id, takerOrder)
		logs = append(logs, openLog)

	} else {
		var remainingSize = takerOrder.Size
		var reason = models.DoneReasonFilled

		if takerOrder.Type == models.OrderTypeMarket {
			takerOrder.Price = decimal.Zero
			remainingSize = decimal.Zero
			if (takerOrder.Side == models.SideSell && takerOrder.Size.GreaterThan(decimal.Zero)) ||
				(takerOrder.Side == models.SideBuy && takerOrder.Funds.GreaterThan(decimal.Zero)) {
				reason = models.DoneReasonCancelled
			}
		}

		doneLog := newDoneLog(o.nextLogSeq(), o.product.Id, takerOrder, remainingSize, reason)
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

	doneLog := newDoneLog(o.nextLogSeq(), o.product.Id, bookOrder, remainingSize, models.DoneReasonCancelled)
	return append(logs, doneLog)
}

func (o *orderBook) Snapshot() orderBookSnapshot {
	snapshot := orderBookSnapshot{
		Orders:        make([]BookOrder, len(o.depths[models.SideSell].orders)+len(o.depths[models.SideBuy].orders)),
		LogSeq:        o.logSeq,
		TradeSeq:      o.tradeSeq,
		OrderIdWindow: o.orderIdWindow,
	}

	i := 0
	for _, order := range o.depths[models.SideSell].orders {
		snapshot.Orders[i] = *order
		i++
	}
	for _, order := range o.depths[models.SideBuy].orders {
		snapshot.Orders[i] = *order
		i++
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

type depth struct {
	// all orders
	orders map[int64]*BookOrder

	// price first, time first order queue for order match
	// priceOrderIdKey -> orderId
	queue *treemap.Map
}

func (d *depth) add(order BookOrder) {
	d.orders[order.OrderId] = &order
	d.queue.Put(&priceOrderIdKey{order.Price, order.OrderId}, order.OrderId)
}

func (d *depth) decrSize(orderId int64, size decimal.Decimal) error {
	order, found := d.orders[orderId]
	if !found {
		return errors.New(fmt.Sprintf("order %v not found on book", orderId))
	}

	if order.Size.LessThan(size) {
		return errors.New(fmt.Sprintf("order %v Size %v less than %v", orderId, order.Size, size))
	}

	order.Size = order.Size.Sub(size)
	if order.Size.IsZero() {
		delete(d.orders, orderId)
		d.queue.Remove(&priceOrderIdKey{order.Price, order.OrderId})
	}

	return nil
}

type BookOrder struct {
	OrderId int64
	Size    decimal.Decimal
	Funds   decimal.Decimal
	Price   decimal.Decimal
	Side    models.Side
	Type    models.OrderType
}

func newBookOrder(order *models.Order) *BookOrder {
	return &BookOrder{
		OrderId: order.Id,
		Size:    order.Size,
		Funds:   order.Funds,
		Price:   order.Price,
		Side:    order.Side,
		Type:    order.Type,
	}
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

	y := aAsserted.orderId - bAsserted.orderId
	if y == 0 {
		return 0
	} else if y > 0 {
		return 1
	} else {
		return -1
	}
}
