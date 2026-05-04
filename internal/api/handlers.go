package api

import (
	"errors"
	"log/slog"
	"net/http"

	"stock-market/internal/domain"
	"stock-market/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Handlers struct {
	Store store.Store
}

var (
	tradesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock_market_trades_total",
			Help: "Total successful trades",
		},
		[]string{"type"},
	)

	tradeErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock_market_trade_errors_total",
			Help: "Total trade errors by reason",
		},
		[]string{"reason"},
	)

	bankSetTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "stock_market_bank_set_total",
			Help: "Total bank resets via POST /stocks",
		},
	)

	walletQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock_market_wallet_queries_total",
			Help: "Total wallet queries by result",
		},
		[]string{"result"},
	)
)

func (h *Handlers) TradeOne(c *gin.Context) {
	walletID := c.Param("wallet_id")
	stockName := c.Param("stock_name")

	var req domain.TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err := h.Store.TradeOne(c.Request.Context(), walletID, stockName, req.Type)
	switch {
	case err == nil:
		tradesTotal.WithLabelValues(req.Type).Inc()
		c.Status(http.StatusOK)

	case errors.Is(err, store.ErrStockNotFound):
		tradeErrorsTotal.WithLabelValues("stock_not_found").Inc()
		slog.Warn("trade_failed", "reason", "stock_not_found", "wallet_id", walletID, "stock", stockName)
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})

	case errors.Is(err, store.ErrBankOutOfStock):
		tradeErrorsTotal.WithLabelValues("bank_out_of_stock").Inc()
		slog.Warn("trade_failed", "reason", "bank_out_of_stock", "wallet_id", walletID, "stock", stockName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "bank out of stock"})

	case errors.Is(err, store.ErrWalletOutOfStock):
		tradeErrorsTotal.WithLabelValues("wallet_out_of_stock").Inc()
		slog.Warn("trade_failed", "reason", "wallet_out_of_stock", "wallet_id", walletID, "stock", stockName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet out of stock"})

	case errors.Is(err, store.ErrInvalidTradeType):
		tradeErrorsTotal.WithLabelValues("invalid_type").Inc()
		slog.Warn("trade_failed", "reason", "invalid_type", "wallet_id", walletID, "stock", stockName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'buy' or 'sell'"})

	default:
		tradeErrorsTotal.WithLabelValues("internal").Inc()
		slog.Error("trade_failed", "reason", "internal", "wallet_id", walletID, "stock", stockName, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handlers) GetWallet(c *gin.Context) {
	walletID := c.Param("wallet_id")
	w, err := h.Store.GetWallet(c.Request.Context(), walletID)
	if err == nil {
		walletQueriesTotal.WithLabelValues("ok").Inc()
		c.JSON(http.StatusOK, w)
		return
	}
	if errors.Is(err, store.ErrWalletNotFound) {
		walletQueriesTotal.WithLabelValues("not_found").Inc()
		slog.Warn("wallet_not_found", "wallet_id", walletID)
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	slog.Error("wallet_query_failed", "wallet_id", walletID, "err", err)
	walletQueriesTotal.WithLabelValues("error").Inc()
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

func (h *Handlers) GetWalletStockQty(c *gin.Context) {
	walletID := c.Param("wallet_id")
	stockName := c.Param("stock_name")

	qty, err := h.Store.GetWalletStockQty(c.Request.Context(), walletID, stockName)
	switch {
	case err == nil:
		c.String(http.StatusOK, "%d", qty)

	case errors.Is(err, store.ErrStockNotFound):
		slog.Warn("stock_not_found", "stock", stockName)
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})

	case errors.Is(err, store.ErrWalletNotFound):
		slog.Warn("wallet_not_found", "wallet_id", walletID)
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})

	default:
		slog.Error("wallet_stock_qty_failed", "wallet_id", walletID, "stock", stockName, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handlers) GetBankStocks(c *gin.Context) {
	stocks, err := h.Store.GetBankStocks(c.Request.Context())
	if err != nil {
		slog.Error("bank_get_failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, domain.BankResponse{Stocks: stocks})
}

func (h *Handlers) SetBankStocks(c *gin.Context) {
	var req domain.SetBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.Store.SetBankStocks(c.Request.Context(), req.Stocks); err != nil {
		slog.Warn("bank_set_failed", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bankSetTotal.Inc()
	c.Status(http.StatusOK)
}

func (h *Handlers) GetAuditLog(c *gin.Context) {
	log, err := h.Store.GetAuditLog(c.Request.Context())
	if err != nil {
		slog.Error("audit_log_failed", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, domain.AuditLogResponse{Log: log})
}
