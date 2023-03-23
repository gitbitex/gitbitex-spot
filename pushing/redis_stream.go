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
	"github.com/gitbitex/gitbitex-spot/conf"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/utils"
	"github.com/go-redis/redis"
	"github.com/siddontang/go-log/log"
	"sync"
	"time"
)

type redisStream struct {
	sub   *subscription
	mutex sync.Mutex
}

func newRedisStream(sub *subscription) *redisStream {
	return &redisStream{
		sub:   sub,
		mutex: sync.Mutex{},
	}
}

func (s *redisStream) Start() {
	gbeConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     gbeConfig.Redis.Addr,
		Password: gbeConfig.Redis.Password,
		DB:       0,
	})
	_, err := redisClient.Ping().Result()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			ps := redisClient.Subscribe(models.TopicOrder)
			_, err := ps.Receive()
			if err != nil {
				log.Error(err)
				continue
			}

			for {
				select {
				case msg := <-ps.Channel():
					var order models.Order
					err := json.Unmarshal([]byte(msg.Payload), &order)
					if err != nil {
						continue
					}

					s.sub.publish(ChannelOrder.Format(order.ProductId, order.UserId), OrderMessage{
						UserId:        order.UserId,
						Type:          "order",
						Sequence:      0,
						Id:            utils.I64ToA(order.Id),
						Price:         order.Price.String(),
						Size:          order.Size.String(),
						Funds:         "0",
						ProductId:     order.ProductId,
						Side:          order.Side.String(),
						OrderType:     order.Type.String(),
						CreatedAt:     order.CreatedAt.Format(time.RFC3339),
						FillFees:      order.FillFees.String(),
						FilledSize:    order.FilledSize.String(),
						ExecutedValue: order.ExecutedValue.String(),
						Status:        order.Status.String(),
						Settled:       order.Settled,
					})
				}
			}
		}
	}()

	go func() {
		for {
			ps := redisClient.Subscribe(models.TopicAccount)
			_, err := ps.Receive()
			if err != nil {
				log.Error(err)
				continue
			}

			for {
				select {
				case msg := <-ps.Channel():
					var account models.Account
					err := json.Unmarshal([]byte(msg.Payload), &account)
					if err != nil {
						continue
					}

					s.sub.publish(ChannelFunds.FormatWithUserId(account.UserId), FundsMessage{
						Type:      "funds",
						Sequence:  0,
						UserId:    utils.I64ToA(account.UserId),
						Currency:  account.Currency,
						Hold:      account.Hold.String(),
						Available: account.Available.String(),
					})
				}
			}
		}
	}()
}
