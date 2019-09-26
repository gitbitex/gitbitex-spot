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
	"github.com/gitbitex/gitbitex-spot/matching"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/utils"
	"github.com/shopspring/decimal"
	"time"
)

type MatchStream struct {
	productId string
	sub       *subscription
	bestBid   decimal.Decimal
	bestAsk   decimal.Decimal
	tick24h   *models.Tick
	tick30d   *models.Tick
	logReader matching.LogReader
}

func newMatchStream(productId string, sub *subscription, logReader matching.LogReader) *MatchStream {
	s := &MatchStream{
		productId: productId,
		sub:       sub,
		logReader: logReader,
	}

	s.logReader.RegisterObserver(s)
	return s
}

func (s *MatchStream) Start() {
	// -1 : read from end
	go s.logReader.Run(0, -1)
}

func (s *MatchStream) OnOpenLog(log *matching.OpenLog, offset int64) {
	// do nothing
}

func (s *MatchStream) OnDoneLog(log *matching.DoneLog, offset int64) {
	// do nothing
}

func (s *MatchStream) OnMatchLog(log *matching.MatchLog, offset int64) {
	// push match
	s.sub.publish(ChannelMatch.FormatWithProductId(log.ProductId), &MatchMessage{
		Type:         "match",
		TradeId:      log.TradeId,
		Sequence:     log.Sequence,
		Time:         log.Time.Format(time.RFC3339),
		ProductId:    log.ProductId,
		Price:        log.Price.String(),
		Side:         log.Side.String(),
		MakerOrderId: utils.I64ToA(log.MakerOrderId),
		TakerOrderId: utils.I64ToA(log.TakerOrderId),
		Size:         log.Size.String(),
	})
}
