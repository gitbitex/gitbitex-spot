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
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gitbitex/gitbitex-spot/conf"
	"github.com/gitbitex/gitbitex-spot/matching"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/service"
	"github.com/gitbitex/gitbitex-spot/utils"
	"github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"github.com/siddontang/go-log/log"
	"net/http"
	"sync"
	"time"
)

var productId2Writer sync.Map

func getWriter(productId string) *kafka.Writer {
	writer, found := productId2Writer.Load(productId)
	if found {
		return writer.(*kafka.Writer)
	}

	gbeConfig, err := conf.GetConfig()
	if err != nil {
		panic(err)
	}
	newWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      gbeConfig.Kafka.Brokers,
		Topic:        matching.TopicOrderPrefix + productId,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 5 * time.Millisecond,
	})
	productId2Writer.Store(productId, newWriter)
	return newWriter
}

func submitOrder(order *models.Order) {
	buf, err := json.Marshal(order)
	if err != nil {
		log.Error(err)
		return
	}

	err = getWriter(order.ProductId).WriteMessages(context.Background(), kafka.Message{Value: buf})
	if err != nil {
		log.Error(err)
	}
}

// POST /orders
func PlaceOrder(ctx *gin.Context) {
	var req placeOrderRequest
	err := ctx.BindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}

	productId := req.ProductId

	side := models.Side(req.Side)
	if len(side) == 0 {
		side = models.SideBuy
	}

	orderType := models.OrderType(req.Type)
	if len(orderType) == 0 {
		orderType = models.OrderTypeLimit
	}

	//todo
	//size, err := utils.StringToFloat64(req.size)
	//price, err := utils.StringToFloat64(req.price)
	size := decimal.NewFromFloat(req.Size)
	price := decimal.NewFromFloat(req.Price)
	funds := decimal.NewFromFloat(req.Funds)

	order, err := service.PlaceOrder(GetCurrentUser(ctx).Id, productId, orderType, side, size, price, funds)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}

	submitOrder(order)

	ctx.JSON(http.StatusOK, OkResponse(order))
}

// 撤销指定id的订单
// DELETE /orders/1
func CancelOrder(ctx *gin.Context) {
	orderIdStr := ctx.Param("orderId")
	orderId, _ := utils.AToInt64(orderIdStr)

	order, err := service.GetOrderById(orderId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	if order == nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse(errors.New("order not found")))
		return
	}

	order.Status = models.OrderStatusCancelling
	submitOrder(order)

	ctx.JSON(http.StatusOK, OkResponse(""))
}

// 批量撤单
// DELETE /orders/?productId=BTC-USDT&side=[buy,sell,nil]&minPrice=1&maxPrice=2
func CancelOrders(ctx *gin.Context) {
	productId := ctx.Query("productId")
	side := ctx.DefaultQuery("side", "")
	if side == "all" {
		side = ""
	}
	//minPrice, _ := decimal.NewFromString(ctx.DefaultQuery("minPrice", "0"))
	//maxPrice, _ := decimal.NewFromString(ctx.DefaultQuery("maxPrice", "99999999999"))

	orders, err := service.GetOrdersByUserId(GetCurrentUser(ctx).Id, []string{string(models.OrderStatusOpen), string(models.OrderStatusNew)}, side, productId, 10000)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}

	for _, order := range orders {
		/*if order.Price.LessThan(minPrice) || order.Price.GreaterThan(maxPrice) {
			continue
		}*/
		order.Status = models.OrderStatusCancelling
		submitOrder(order)
	}

	ctx.JSON(http.StatusOK, OkResponse(""))
}

// GET /orders
func GetOrders(ctx *gin.Context) {
	productId := ctx.Query("productId")
	user := GetCurrentUser(ctx)
	if user == nil {
		ctx.JSON(http.StatusForbidden, ErrorResponse(errors.New("current user not present")))
		return
	}

	orderVos := []*orderVo{}
	orders, err := service.GetOrdersByUserId(user.Id,
		[]string{string(models.OrderStatusNew), string(models.OrderStatusOpen)}, "", productId, 100)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse(err))
		return
	}
	for _, order := range orders {
		orderVos = append(orderVos, order2OrderVo(order))
	}

	ctx.JSON(http.StatusOK, OkResponse(orderVos))
}
