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
	"fmt"
	"github.com/gitbitex/gitbitex-spot/matching"
	logger "github.com/siddontang/go-log/log"
	"sync"
	"time"
)

type OrderBookStream struct {
	productId  string
	logReader  matching.LogReader
	logCh      chan *logOffset
	orderBook  *orderBook
	sub        *subscription
	snapshotCh chan interface{}
}

type logOffset struct {
	log    interface{}
	offset int64
}

func newOrderBookStream(productId string, sub *subscription, logReader matching.LogReader) *OrderBookStream {
	s := &OrderBookStream{
		productId:  productId,
		orderBook:  newOrderBook(productId),
		logCh:      make(chan *logOffset, 1000),
		sub:        sub,
		logReader:  logReader,
		snapshotCh: make(chan interface{}, 100),
	}

	// try restore snapshot
	snapshot, err := sharedSnapshotStore().getLastFull(productId)
	if err != nil {
		logger.Fatalf("get snapshot error: %v", err)
	}
	if snapshot != nil {
		s.orderBook.Restore(snapshot)
		logger.Infof("%v order book snapshot loaded: %+v", s.productId, snapshot)
	}

	s.logReader.RegisterObserver(s)
	return s
}

func (s *OrderBookStream) Start() {
	logOffset := s.orderBook.logOffset
	if logOffset > 0 {
		logOffset++
	}
	go s.logReader.Run(s.orderBook.logSeq, logOffset)
	go s.runApplier()
	go s.runSnapshots()
}

func (s *OrderBookStream) OnOpenLog(log *matching.OpenLog, offset int64) {
	s.logCh <- &logOffset{log, offset}
}

func (s *OrderBookStream) OnMatchLog(log *matching.MatchLog, offset int64) {
	s.logCh <- &logOffset{log, offset}
}

func (s *OrderBookStream) OnDoneLog(log *matching.DoneLog, offset int64) {
	s.logCh <- &logOffset{log, offset}
}

func (s *OrderBookStream) runApplier() {
	var lastLevel2Snapshot *OrderBookLevel2Snapshot
	var lastFullSnapshot *OrderBookFullSnapshot

	for {
		select {
		case logOffset := <-s.logCh:
			var l2Change *Level2Change

			switch logOffset.log.(type) {
			case *matching.DoneLog:
				log := logOffset.log.(*matching.DoneLog)
				order, found := s.orderBook.orders[log.OrderId]
				if !found {
					continue
				}
				newSize := order.Size.Sub(log.RemainingSize)
				l2Change = s.orderBook.saveOrder(logOffset.offset, log.Sequence, log.OrderId, newSize, log.Price,
					log.Side)

			case *matching.OpenLog:
				log := logOffset.log.(*matching.OpenLog)
				l2Change = s.orderBook.saveOrder(logOffset.offset, log.Sequence, log.OrderId, log.RemainingSize,
					log.Price, log.Side)

			case *matching.MatchLog:
				log := logOffset.log.(*matching.MatchLog)
				order, found := s.orderBook.orders[log.MakerOrderId]
				if !found {
					panic(fmt.Sprintf("should not happen : %+v", log))
				}
				newSize := order.Size.Sub(log.Size)
				l2Change = s.orderBook.saveOrder(logOffset.offset, log.Sequence, log.MakerOrderId, newSize,
					log.Price, log.Side)
			}

			if lastLevel2Snapshot == nil || s.orderBook.seq-lastLevel2Snapshot.Seq > 10 {
				lastLevel2Snapshot = s.orderBook.SnapshotLevel2(1000)
				lastLevel2Snapshots.Store(s.productId, lastLevel2Snapshot)
			}

			if lastFullSnapshot == nil || s.orderBook.seq-lastFullSnapshot.Seq > 10000 {
				lastFullSnapshot = s.orderBook.SnapshotFull()
				s.snapshotCh <- lastFullSnapshot
			}

			if l2Change != nil {
				s.sub.publish(ChannelLevel2.FormatWithProductId(s.productId), l2Change)
			}

		case <-time.After(200 * time.Millisecond):
			if lastLevel2Snapshot == nil || s.orderBook.seq > lastLevel2Snapshot.Seq {
				lastLevel2Snapshot = s.orderBook.SnapshotLevel2(1000)
				lastLevel2Snapshots.Store(s.productId, lastLevel2Snapshot)
			}
		}
	}
}

func (s *OrderBookStream) runSnapshots() {
	for {
		select {
		case snapshot := <-s.snapshotCh:
			switch snapshot.(type) {
			case *OrderBookLevel2Snapshot:
				err := sharedSnapshotStore().storeLevel2(s.productId, snapshot.(*OrderBookLevel2Snapshot))
				if err != nil {
					logger.Error(err)
				}
			case *OrderBookFullSnapshot:
				err := sharedSnapshotStore().storeFull(s.productId, snapshot.(*OrderBookFullSnapshot))
				if err != nil {
					logger.Error(err)
				}
			}
		}
	}
}

var lastLevel2Snapshots = sync.Map{}

func getLastLevel2Snapshot(productId string) *OrderBookLevel2Snapshot {
	snapshot, found := lastLevel2Snapshots.Load(productId)
	if !found {
		return nil
	}
	return snapshot.(*OrderBookLevel2Snapshot)
}
