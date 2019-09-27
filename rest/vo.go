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

package rest

import (
	"fmt"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/utils"
	"strings"
	"time"
)

type messageVo struct {
	Message string `json:"message"`
}

func newMessageVo(error error) *messageVo {
	return &messageVo{
		Message: error.Error(),
	}
}

type AccountVo struct {
	Id           string `json:"id"`
	Currency     string `json:"currency"`
	CurrencyIcon string `json:"currencyIcon"`
	Available    string `json:"available"`
	Hold         string `json:"hold"`
}

type placeOrderRequest struct {
	ClientOid   string  `json:"client_oid"`
	ProductId   string  `json:"productId"`
	Size        float64 `json:"size"`
	Funds       float64 `json:"funds"`
	Price       float64 `json:"price"`
	Side        string  `json:"side"`
	Type        string  `json:"type"`        // [optional] limit or market (default is limit)
	TimeInForce string  `json:"timeInForce"` // [optional] GTC, GTT, IOC, or FOK (default is GTC)
}

type orderVo struct {
	Id            string `json:"id"`
	Price         string `json:"price"`
	Size          string `json:"size"`
	Funds         string `json:"funds"`
	ProductId     string `json:"productId"`
	Side          string `json:"side"`
	Type          string `json:"type"`
	CreatedAt     string `json:"createdAt"`
	FillFees      string `json:"fillFees"`
	FilledSize    string `json:"filledSize"`
	ExecutedValue string `json:"executedValue"`
	Status        string `json:"status"`
	Settled       bool   `json:"settled"`
}

const (
	Level1 = "1"
	Level2 = "2"
	Level3 = "3"
)

type ProductVo struct {
	Id             string `json:"id"`
	BaseCurrency   string `json:"baseCurrency"`
	QuoteCurrency  string `json:"quoteCurrency"`
	BaseMinSize    string `json:"baseMinSize"`
	BaseMaxSize    string `json:"baseMaxSize"`
	QuoteIncrement string `json:"quoteIncrement"`
	BaseScale      int32  `json:"baseScale"`
	QuoteScale     int32  `json:"quoteScale"`
}

type tradeVo struct {
	Time    string `json:"time"`
	TradeId int64  `json:"tradeId"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Side    string `json:"side"`
}

type orderBookVo struct {
	Sequence string           `json:"sequence"`
	Asks     [][3]interface{} `json:"asks"`
	Bids     [][3]interface{} `json:"bids"`
}

type SignUpRequest struct {
	Email    string
	Password string
}

type changePasswordRequest struct {
	OldPassword string
	NewPassword string
}

type userVo struct {
	Id           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	ProfilePhoto string `json:"profilePhoto"`
	IsBand       bool   `json:"isBand"`
	CreatedAt    string `json:"createdAt"`
}

type walletAddressVo struct {
	Address string `json:"address"`
}

type networkVo struct {
	Status        string `json:"status"`
	Hash          string `json:"hash"`
	Amount        string `json:"amount"`
	FeeAmount     string `json:"feeAmount"`
	FeeCurrency   string `json:"feeCurrency"`
	Confirmations int    `json:"confirmations"`
	ResourceUrl   string `json:"resourceUrl"`
}

type transactionVo struct {
	Id             string    `json:"id"`
	Currency       string    `json:"currency"`
	Amount         string    `json:"amount"`
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	Price          string    `json:"price"`
	NativeCurrency string    `json:"nativeCurrency"`
	NativeAmount   string    `json:"nativeAmount"`
	Description    string    `json:"description"`
	CreatedAt      string    `json:"createdAt"`
	UpdatedAt      string    `json:"updatedAt"`
	FromAddress    string    `json:"fromAddress"`
	ToAddress      string    `json:"toAddress"`
	Network        networkVo `json:"network"`
}

func newTransactionVo() *transactionVo {
	return &transactionVo{
		Id:             "1",
		Currency:       "BTC",
		Amount:         "0.1",
		Type:           "send",
		Status:         "pending",
		Price:          "8000",
		NativeCurrency: "USD",
		NativeAmount:   "8000",
		Description:    "withdraw btc",
		CreatedAt:      time.Now().Format(time.RFC3339),
		UpdatedAt:      time.Now().Format(time.RFC3339),
		FromAddress:    "0x3A3E32423AE323242",
		ToAddress:      "0x4B3E32423AE323242",
		Network: networkVo{
			Status:        "confirmed",
			Hash:          "0x23423AA3232",
			Amount:        "0.1",
			FeeAmount:     "0.1",
			FeeCurrency:   "BTC",
			Confirmations: 0,
			ResourceUrl:   "",
		},
	}
}

func newTradeVo(trade *models.Trade) *tradeVo {
	return &tradeVo{
		Time:    trade.Time.Format(time.RFC3339),
		TradeId: trade.Id,
		Price:   trade.Price.String(),
		Size:    trade.Size.String(),
		Side:    trade.Side.String(),
	}
}

func newProductVo(product *models.Product) *ProductVo {
	return &ProductVo{
		Id:             product.Id,
		BaseCurrency:   product.BaseCurrency,
		QuoteCurrency:  product.QuoteCurrency,
		BaseMinSize:    product.BaseMinSize.String(),
		BaseMaxSize:    product.BaseMaxSize.String(),
		QuoteIncrement: utils.F64ToA(product.QuoteIncrement),
		BaseScale:      product.BaseScale,
		QuoteScale:     product.QuoteScale,
	}
}

func newOrderVo(order *models.Order) *orderVo {
	return &orderVo{
		Id:            utils.I64ToA(order.Id),
		Price:         order.Price.String(),
		Size:          order.Size.String(),
		Funds:         order.ExecutedValue.String(),
		ProductId:     order.ProductId,
		Side:          order.Side.String(),
		Type:          order.Type.String(),
		CreatedAt:     order.CreatedAt.Format(time.RFC3339),
		FillFees:      order.FillFees.String(),
		FilledSize:    order.FilledSize.String(),
		ExecutedValue: order.ExecutedValue.String(),
		Status:        order.Status.String(),
		Settled:       order.Settled,
	}
}

func newAccountVo(account *models.Account) *AccountVo {
	return &AccountVo{
		Id:           fmt.Sprintf("%v", account.Id),
		Currency:     account.Currency,
		CurrencyIcon: fmt.Sprintf("https://oooooo.oss-cn-hangzhou.aliyuncs.com/coin/%v.png", strings.ToLower(account.Currency)),
		Available:    account.Available.String(),
		Hold:         account.Hold.String(),
	}
}
