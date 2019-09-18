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

package service

import (
	"errors"
	"fmt"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/models/mysql"
	"github.com/shopspring/decimal"
)

func ExecuteBill(userId int64, currency string) error {
	tx, err := mysql.SharedStore().BeginTx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 锁定用户资金记录
	account, err := tx.GetAccountForUpdate(userId, currency)
	if err != nil {
		return err
	}
	// 资金记录不存在，创建一条，并再次执行加锁
	if account == nil {
		err = tx.AddAccount(&models.Account{
			UserId:    userId,
			Currency:  currency,
			Available: decimal.Zero,
		})
		if err != nil {
			return err
		}
		account, err = tx.GetAccountForUpdate(userId, currency)
		if err != nil {
			return err
		}
	}

	// 获取所有未入账的bill
	bills, err := tx.GetUnsettledBillsByUserId(userId, currency)
	if err != nil {
		return err
	}
	if len(bills) == 0 {
		return nil
	}

	for _, bill := range bills {
		account.Available = account.Available.Add(bill.Available)
		account.Hold = account.Hold.Add(bill.Hold)

		bill.Settled = true

		err = tx.UpdateBill(bill)
		if err != nil {
			return err
		}
	}

	err = tx.UpdateAccount(account)
	if err != nil {
		return err
	}

	err = tx.CommitTx()
	if err != nil {
		return err
	}

	return nil
}

func HoldBalance(db models.Store, userId int64, currency string, size decimal.Decimal, billType models.BillType) error {
	if size.LessThanOrEqual(decimal.Zero) {
		return errors.New("size less than 0")
	}

	enough, err := HasEnoughBalance(userId, currency, size)
	if err != nil {
		return err
	}
	if !enough {
		return errors.New(fmt.Sprintf("no enough %v : request=%v", currency, size))
	}

	account, err := db.GetAccountForUpdate(userId, currency)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("no enough")
	}

	account.Available = account.Available.Sub(size)
	account.Hold = account.Hold.Add(size)

	bill := &models.Bill{
		UserId:    userId,
		Currency:  currency,
		Available: size.Neg(),
		Hold:      size,
		Type:      billType,
		Settled:   true,
		Notes:     "",
	}
	err = db.AddBills([]*models.Bill{bill})
	if err != nil {
		return err
	}

	err = db.UpdateAccount(account)
	if err != nil {
		return err
	}

	return nil
}

func HasEnoughBalance(userId int64, currency string, size decimal.Decimal) (bool, error) {
	account, err := GetAccount(userId, currency)
	if err != nil {
		return false, err
	}
	if account == nil {
		return false, nil
	}
	return account.Available.GreaterThanOrEqual(size), nil
}

func GetAccount(userId int64, currency string) (*models.Account, error) {
	return mysql.SharedStore().GetAccount(userId, currency)
}

func GetAccountsByUserId(userId int64) ([]*models.Account, error) {
	return mysql.SharedStore().GetAccountsByUserId(userId)
}

func AddDelayBill(store models.Store, userId int64, currency string, available, hold decimal.Decimal, billType models.BillType, notes string) (*models.Bill, error) {
	bill := &models.Bill{
		UserId:    userId,
		Currency:  currency,
		Available: available,
		Hold:      hold,
		Type:      billType,
		Settled:   false,
		Notes:     notes,
	}
	err := store.AddBills([]*models.Bill{bill})
	return bill, err
}

func GetUnsettledBills() ([]*models.Bill, error) {
	return mysql.SharedStore().GetUnsettledBills()
}
