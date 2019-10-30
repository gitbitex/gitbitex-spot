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

package mysql

import (
	"fmt"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/jinzhu/gorm"
	"strings"
)

func (s *Store) GetLastFillByProductId(productId string) (*models.Fill, error) {
	var fill models.Fill
	err := s.db.Where("product_id =?", productId).Order("id DESC").Limit(1).Find(&fill).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &fill, err
}

func (s *Store) GetUnsettledFillsByOrderId(orderId int64) ([]*models.Fill, error) {
	db := s.db.Where("settled =?", 0).Where("order_id=?", orderId).
		Order("id ASC").Limit(100)

	var fills []*models.Fill
	err := db.Find(&fills).Error
	return fills, err
}

func (s *Store) GetUnsettledFills(count int32) ([]*models.Fill, error) {
	db := s.db.Where("settled =?", 0).Order("id ASC").Limit(count)

	var fills []*models.Fill
	err := db.Find(&fills).Error
	return fills, err
}

func (s *Store) UpdateFill(fill *models.Fill) error {
	return s.db.Save(fill).Error
}

func (s *Store) AddFills(fills []*models.Fill) error {
	if len(fills) == 0 {
		return nil
	}
	var valueStrings []string
	for _, fill := range fills {
		valueString := fmt.Sprintf("(NOW(), '%v', %v, %v, %v, %v,%v, %v,'%v',%v,%v,'%v',%v,'%v',%v,%v)",
			fill.ProductId, fill.TradeId, fill.OrderId, fill.MessageSeq, fill.Size, fill.Price, fill.Funds,
			fill.Liquidity, fill.Fee, fill.Settled, fill.Side, fill.Done, fill.DoneReason, fill.LogOffset, fill.LogSeq)
		valueStrings = append(valueStrings, valueString)
	}
	sql := fmt.Sprintf("INSERT IGNORE INTO g_fill (created_at,product_id,trade_id,order_id, message_seq,size,"+
		"price,funds,liquidity,fee,settled,side,done,done_reason,log_offset,log_seq) VALUES %s",
		strings.Join(valueStrings, ","))
	return s.db.Exec(sql).Error
}
