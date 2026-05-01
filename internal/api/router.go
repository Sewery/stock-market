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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RouterConfig struct {
	LogHTTP      bool
	LogRequestID bool
}

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock_market_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stock_market_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)
)

func NewRouter(h *Handlers, cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(requestIDMiddleware(cfg.LogRequestID))
	r.Use(httpLoggerMiddleware(cfg.LogHTTP))

	r.Use(httpMetricsMiddleware())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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

func httpMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		httpRequestsTotal.WithLabelValues(path, c.Request.Method, fmt.Sprintf("%d", c.Writer.Status())).Inc()
		httpRequestDuration.WithLabelValues(path, c.Request.Method).Observe(time.Since(start).Seconds())
	}
}
