package api

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func NewRouter(h *Handlers) *gin.Engine {
	r := gin.Default()

	r.POST("/wallets/:wallet_id/stocks/:stock_name", h.TradeOne)
	r.GET("/wallets/:wallet_id", h.GetWallet)
	r.GET("/wallets/:wallet_id/stocks/:stock_name", h.GetWalletStockQty)

	r.GET("/stocks", h.GetBankStocks)
	r.POST("/stocks", h.SetBankStocks)

	r.GET("/log", h.GetAuditLog)

	r.POST("/chaos", func(c *gin.Context) {
		c.Status(http.StatusOK)
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(1)
		}()
	})

	return r
}
