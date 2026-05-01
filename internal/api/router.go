package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type RouterConfig struct {
	LogHTTP      bool
	LogRequestID bool
}

func NewRouter(h *Handlers, cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(requestIDMiddleware(cfg.LogRequestID))
	r.Use(httpLoggerMiddleware(cfg.LogHTTP))

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

func requestIDMiddleware(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		c.Writer.Header().Set("X-Request-ID", id)
		c.Set("request_id", id)
		c.Next()
	}
}

func httpLoggerMiddleware(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		}

		if v, ok := c.Get("request_id"); ok {
			attrs = append(attrs, "request_id", v)
		}

		slog.Info("http_request", attrs...)
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
