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
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/jinzhu/gorm"
	"time"
)

func (s *Store) GetOrderById(orderId int64) (*models.Order, error) {
	var order models.Order
	err := s.db.Raw("SELECT * FROM g_order WHERE id=?", orderId).Scan(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &order, err
}

func (s *Store) GetOrderByClientOid(userId int64, clientOid string) (*models.Order, error) {
	var order models.Order
	err := s.db.Raw("SELECT * FROM g_order WHERE user_id=? AND client_oid=?", userId, clientOid).Scan(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &order, err
}

func (s *Store) GetOrderByIdForUpdate(orderId int64) (*models.Order, error) {
	var order models.Order
	err := s.db.Raw("SELECT * FROM g_order WHERE id=? FOR UPDATE", orderId).Scan(&order).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &order, err
}

func (s *Store) GetOrdersByUserId(userId int64, statuses []models.OrderStatus, side *models.Side, productId string,
	beforeId, afterId int64, limit int) ([]*models.Order, error) {
	db := s.db.Where("user_id =?", userId)

	if len(statuses) != 0 {
		db = db.Where("status IN (?)", statuses)
	}

	if len(productId) != 0 {
		db = db.Where("product_id=?", productId)
	}

	if side != nil {
		db = db.Where("side=?", side)
	}

	if beforeId > 0 {
		db = db.Where("id>?", beforeId)
	}

	if afterId > 0 {
		db = db.Where("id<?", afterId)
	}

	if limit <= 0 {
		limit = 100
	}

	db = db.Order("id DESC").Limit(limit)

	var orders []*models.Order
	err := db.Find(&orders).Error
	return orders, err
}

func (s *Store) AddOrder(order *models.Order) error {
	order.CreatedAt = time.Now()
	return s.db.Create(order).Error
}

func (s *Store) UpdateOrder(order *models.Order) error {
	order.UpdatedAt = time.Now()
	return s.db.Save(order).Error
}

func (s *Store) UpdateOrderStatus(orderId int64, oldStatus, newStatus models.OrderStatus) (bool, error) {
	ret := s.db.Exec("UPDATE g_order SET `status`=?,updated_at=? WHERE id=? AND `status`=? ", newStatus, time.Now(), orderId, oldStatus)
	if ret.Error != nil {
		return false, ret.Error
	}
	return ret.RowsAffected > 0, nil
}
