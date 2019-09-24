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

package models

type Store interface {
	BeginTx() (Store, error)
	Rollback() error
	CommitTx() error

	GetConfigs() ([]*Config, error)

	GetUserByEmail(email string) (*User, error)
	AddUser(user *User) error
	UpdateUser(user *User) error

	GetAccount(userId int64, currency string) (*Account, error)
	GetAccountsByUserId(userId int64) ([]*Account, error)
	GetAccountForUpdate(userId int64, currency string) (*Account, error)
	AddAccount(account *Account) error
	UpdateAccount(account *Account) error

	GetUnsettledBillsByUserId(userId int64, currency string) ([]*Bill, error)
	GetUnsettledBills() ([]*Bill, error)
	AddBills(bills []*Bill) error
	UpdateBill(bill *Bill) error

	GetProductById(id string) (*Product, error)
	GetProducts() ([]*Product, error)

	GetOrderById(orderId int64) (*Order, error)
	GetOrderByClientOid(userId int64, clientOid string) (*Order, error)
	GetOrderByIdForUpdate(orderId int64) (*Order, error)
	GetOrdersByUserId(userId int64, statuses []OrderStatus, side *Side, productId string,
		beforeId, afterId int64, limit int) ([]*Order, error)
	AddOrder(order *Order) error
	UpdateOrder(order *Order) error
	UpdateOrderStatus(orderId int64, oldStatus, newStatus OrderStatus) (bool, error)

	GetLastFillByProductId(productId string) (*Fill, error)
	GetUnsettledFillsByOrderId(orderId int64) ([]*Fill, error)
	GetUnsettledFills(count int32) ([]*Fill, error)
	UpdateFill(fill *Fill) error
	AddFills(fills []*Fill) error

	GetLastTradeByProductId(productId string) (*Trade, error)
	GetTradesByProductId(productId string, count int) ([]*Trade, error)
	AddTrades(trades []*Trade) error

	GetTicksByProductId(productId string, granularity int64, limit int) ([]*Tick, error)
	GetLastTickByProductId(productId string, granularity int64) (*Tick, error)
	AddTicks(ticks []*Tick) error
}
