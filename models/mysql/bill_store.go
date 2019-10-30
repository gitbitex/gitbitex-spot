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
	"strings"
	"time"
)

func (s *Store) GetUnsettledBillsByUserId(userId int64, currency string) ([]*models.Bill, error) {
	db := s.db.Where("settled =?", 0).Where("user_id=?", userId).
		Where("currency=?", currency).Order("id ASC").Limit(100)

	var bills []*models.Bill
	err := db.Find(&bills).Error
	return bills, err
}

func (s *Store) GetUnsettledBills() ([]*models.Bill, error) {
	db := s.db.Where("settled =?", 0).Order("id ASC").Limit(100)

	var bills []*models.Bill
	err := db.Find(&bills).Error
	return bills, err
}

func (s *Store) AddBills(bills []*models.Bill) error {
	if len(bills) == 0 {
		return nil
	}
	var valueStrings []string
	for _, bill := range bills {
		valueString := fmt.Sprintf("(NOW(),%v, '%v', %v, %v, '%v', %v, '%v')",
			bill.UserId, bill.Currency, bill.Available, bill.Hold, bill.Type, bill.Settled, bill.Notes)
		valueStrings = append(valueStrings, valueString)
	}
	sql := fmt.Sprintf("INSERT INTO g_bill (created_at, user_id,currency,available,hold, type,settled,notes) VALUES %s", strings.Join(valueStrings, ","))
	return s.db.Exec(sql).Error
}

func (s *Store) UpdateBill(bill *models.Bill) error {
	bill.UpdatedAt = time.Now()
	return s.db.Save(bill).Error
}
