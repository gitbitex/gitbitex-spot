package rest

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// GET /wallets/{currency}/address
func GetWalletAddress(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, walletAddressVo{Address: "Coming Soon"})
}

// GET /wallets/{currency}/transactions
func GetWalletTransactions(ctx *gin.Context) {
	currency := ctx.Param("currency")

	transactionVos := []*transactionVo{}
	if currency == "BTC" {
		transactionVos = append(transactionVos, newTransactionVo())
	}

	ctx.JSON(http.StatusOK, transactionVos)
}

// POST /wallets/{currency}/withdrawal
func Withdrawal(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, transactionVo{
		Id:       "1",
		Currency: "BTC",
		Amount:   "0.1",
	})
}
