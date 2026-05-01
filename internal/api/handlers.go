package api

import (
	"errors"
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
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})

	case errors.Is(err, store.ErrBankOutOfStock):
		tradeErrorsTotal.WithLabelValues("bank_out_of_stock").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "bank out of stock"})

	case errors.Is(err, store.ErrWalletOutOfStock):
		tradeErrorsTotal.WithLabelValues("wallet_out_of_stock").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet out of stock"})

	case errors.Is(err, store.ErrInvalidTradeType):
		tradeErrorsTotal.WithLabelValues("invalid_type").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'buy' or 'sell'"})

	default:
		tradeErrorsTotal.WithLabelValues("internal").Inc()
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
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
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
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
	case errors.Is(err, store.ErrWalletNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handlers) GetBankStocks(c *gin.Context) {
	stocks, err := h.Store.GetBankStocks(c.Request.Context())
	if err != nil {
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
		// validation errors should be 400; store implementations return plain errors
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bankSetTotal.Inc()
	c.Status(http.StatusOK)
}

func (h *Handlers) GetAuditLog(c *gin.Context) {
	log, err := h.Store.GetAuditLog(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, domain.AuditLogResponse{Log: log})
}
