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

import (
	"github.com/shopspring/decimal"
	"time"
)

type OrderType string
type Side string
type OrderStatus string
type BillType string
type DoneReason string

const (
	OrderTypeLimit  = OrderType("limit")
	OrderTypeMarket = OrderType("market")

	SideBuy  = Side("buy")
	SideSell = Side("sell")

	// 初始状态
	OrderStatusNew = OrderStatus("new")
	// 已经加入orderBook
	OrderStatusOpen       = OrderStatus("open")
	OrderStatusCancelling = OrderStatus("cancelling")
	OrderStatusCancelled  = OrderStatus("cancelled")
	OrderStatusFilled     = OrderStatus("filled")

	BillTypeTrade = BillType("trade")

	DoneReasonFilled    = DoneReason("filled")
	DoneReasonCancelled = DoneReason("cancelled")
	DoneReasonRejected  = DoneReason("rejected")
)

type User struct {
	Id           int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	UserId       int64
	Email        string
	PasswordHash string
}

type Account struct {
	Id        int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt time.Time
	UpdatedAt time.Time
	UserId    int64           `gorm:"column:user_id;unique_index:idx_uid_currency"`
	Currency  string          `gorm:"column:currency;unique_index:idx_uid_currency"`
	Hold      decimal.Decimal `gorm:"column:hold" sql:"type:decimal(32,16);"`
	Available decimal.Decimal `gorm:"column:available" sql:"type:decimal(32,16);"`
}

type Bill struct {
	Id        int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt time.Time
	UpdatedAt time.Time
	UserId    int64
	Currency  string
	Available decimal.Decimal `sql:"type:decimal(32,16);"`
	Hold      decimal.Decimal `sql:"type:decimal(32,16);"`
	Type      BillType
	Settled   bool
	Notes     string
}

type Product struct {
	Id             string `gorm:"column:id;primary_key"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	BaseCurrency   string
	QuoteCurrency  string
	BaseMinSize    decimal.Decimal `sql:"type:decimal(32,16);"`
	BaseMaxSize    decimal.Decimal `sql:"type:decimal(32,16);"`
	QuoteMinSize   decimal.Decimal `sql:"type:decimal(32,16);"`
	QuoteMaxSize   decimal.Decimal `sql:"type:decimal(32,16);"`
	BaseScale      int32
	QuoteScale     int32
	QuoteIncrement float64
}

type Order struct {
	Id            int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ProductId     string
	UserId        int64
	Size          decimal.Decimal `sql:"type:decimal(32,16);"`
	Funds         decimal.Decimal `sql:"type:decimal(32,16);"`
	FilledSize    decimal.Decimal `sql:"type:decimal(32,16);"`
	ExecutedValue decimal.Decimal `sql:"type:decimal(32,16);"`
	Price         decimal.Decimal `sql:"type:decimal(32,16);"`
	FillFees      decimal.Decimal `sql:"type:decimal(32,16);"`
	Type          OrderType
	Side          Side
	TimeInForce   string
	Status        OrderStatus
	Settled       bool
}

type Fill struct {
	Id         int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	TradeId    int64
	OrderId    int64 `gorm:"unique_index:o_m"`
	MessageSeq int64 `gorm:"unique_index:o_m"`
	ProductId  string
	Size       decimal.Decimal `sql:"type:decimal(32,16);"`
	Price      decimal.Decimal `sql:"type:decimal(32,16);"`
	Funds      decimal.Decimal `sql:"type:decimal(32,16);"`
	Fee        decimal.Decimal `sql:"type:decimal(32,16);"`
	Liquidity  string
	Settled    bool
	Side       Side
	Done       bool
	DoneReason DoneReason
	LogOffset  int64
	LogSeq     int64
}

type Trade struct {
	Id           int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ProductId    string
	TakerOrderId int64
	MakerOrderId int64
	Price        decimal.Decimal `sql:"type:decimal(32,16);"`
	Size         decimal.Decimal `sql:"type:decimal(32,16);"`
	Side         Side
	Time         time.Time
	LogOffset    int64
	LogSeq       int64
}

type Tick struct {
	Id          int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProductId   string          `gorm:"unique_index:p_g_t"`
	Granularity int64           `gorm:"unique_index:p_g_t"`
	Time        int64           `gorm:"unique_index:p_g_t"`
	Open        decimal.Decimal `sql:"type:decimal(32,16);"`
	High        decimal.Decimal `sql:"type:decimal(32,16);"`
	Low         decimal.Decimal `sql:"type:decimal(32,16);"`
	Close       decimal.Decimal `sql:"type:decimal(32,16);"`
	Volume      decimal.Decimal `sql:"type:decimal(32,16);"`
	LogOffset   int64
	LogSeq      int64
}

type Config struct {
	Id        int64 `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Key       string
	Value     string
}

func (s Side) Opposite() Side {
	if s == SideBuy {
		return SideSell
	}
	return SideBuy
}

func (s Side) String() string {
	return string(s)
}

func (t OrderType) String() string {
	return string(t)
}

func (t OrderStatus) String() string {
	return string(t)
}
