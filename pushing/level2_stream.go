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
	"time"
)

type OrderBookStream struct {
	productId string
	logReader matching.LogReader
	logCh     chan *logOffset
	orderBook *orderBook
	sub       *subscription
}

type logOffset struct {
	log    interface{}
	offset int64
}

func newOrderBookStream(productId string, sub *subscription, logReader matching.LogReader) *OrderBookStream {
	s := &OrderBookStream{
		productId: productId,
		orderBook: newOrderBook(productId),
		logCh:     make(chan *logOffset, 1000),
		sub:       sub,
		logReader: logReader,
	}

	// 恢复snapshot
	snapshot, err := sharedSnapshotStore().getLastFull(productId)
	if err != nil {
		panic(err)
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
	go s.logReader.Run("level2Stream", s.orderBook.logSeq, logOffset)
	go s.flush()
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

func (s *OrderBookStream) flush() {
	var lastLevel2Snapshot OrderBookLevel2Snapshot
	var lastFullSnapshot OrderBookFullSnapshot

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

			if l2Change != nil {
				s.sub.publish(ChannelLevel2.FormatWithProductId(s.productId), l2Change)
			}

			delta := s.orderBook.seq - lastLevel2Snapshot.Seq
			if delta > 10 {
				lastLevel2Snapshot = s.orderBook.SnapshotLevel2()
				err := sharedSnapshotStore().storeLevel2(s.productId, &lastLevel2Snapshot)
				if err != nil {
					logger.Error(err)
				}
			}

			delta = s.orderBook.seq - lastFullSnapshot.Seq
			if delta > 10 {
				lastFullSnapshot = s.orderBook.SnapshotFull()
				err := sharedSnapshotStore().storeFull(s.productId, &lastFullSnapshot)
				if err != nil {
					logger.Error(err)
				}
			}

		case <-time.After(200 * time.Millisecond):
			if s.orderBook.seq > lastLevel2Snapshot.Seq {
				lastLevel2Snapshot = s.orderBook.SnapshotLevel2()
				err := sharedSnapshotStore().storeLevel2(s.productId, &lastLevel2Snapshot)
				if err != nil {
					logger.Error(err)
				}
			}

			if s.orderBook.seq > lastLevel2Snapshot.Seq {
				lastFullSnapshot = s.orderBook.SnapshotFull()
				err := sharedSnapshotStore().storeFull(s.productId, &lastFullSnapshot)
				if err != nil {
					logger.Error(err)
				}
			}
		}
	}
}
