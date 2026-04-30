package main

import (
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type StockQty struct {
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type BankResponse struct {
	Stocks []StockQty `json:"stocks"`
}

type SetBankRequest struct {
	Stocks []StockQty `json:"stocks"`
}

type TradeRequest struct {
	Type string `json:"type"` // "buy" | "sell"
}

type WalletResponse struct {
	ID     string     `json:"id"`
	Stocks []StockQty `json:"stocks"`
}

type AuditLogEntry struct {
	Type      string `json:"type"` // "buy" | "sell"
	WalletID  string `json:"wallet_id"`
	StockName string `json:"stock_name"`
}

type AuditLogResponse struct {
	Log []AuditLogEntry `json:"log"`
}

type MarketState struct {
	mu sync.RWMutex

	// Bank is just a map of stocks, transactions
	bank map[string]int64 // stock needs to have qty >= 0

	// wallet exists, if is a key
	walletStocks map[string]map[string]int64 // wallet to (stock to qty >=0)

	// logging only successful BUY/SELL
	audit []AuditLogEntry
}

func main() {
	state := &MarketState{
		bank:         map[string]int64{},
		walletStocks: map[string]map[string]int64{},
		audit:        make([]AuditLogEntry, 0, 10_000),
	}

	r := gin.Default()

	r.POST("/stocks", func(c *gin.Context) { setBankStocks(c, state) })
	r.GET("/stocks", func(c *gin.Context) { getBankStocks(c, state) })

	r.POST("/wallets/:wallet_id/stocks/:stock_name", func(c *gin.Context) { tradeOne(c, state) })
	r.GET("/wallets/:wallet_id", func(c *gin.Context) { getWallet(c, state) })
	r.GET("/wallets/:wallet_id/stocks/:stock_name", func(c *gin.Context) { getWalletStockQty(c, state) })

	r.GET("/log", func(c *gin.Context) { getAuditLog(c, state) })

	r.POST("/chaos", func(c *gin.Context) {
		c.Status(http.StatusOK)
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(1)
		}()
	})

	host := getEnvDefault("HOST", "localhost")
	port := getEnvDefault("PORT", "8080")
	_ = r.Run(host + ":" + port)
}

func setBankStocks(c *gin.Context, state *MarketState) {
	var req SetBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	newBank := make(map[string]int64, len(req.Stocks))
	for _, st := range req.Stocks {
		if st.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "stock name is required"})
			return
		}
		if st.Quantity < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "stock quantity must be >= 0"})
			return
		}
		if _, dup := newBank[st.Name]; dup {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate stock name: " + st.Name})
			return
		}
		newBank[st.Name] = st.Quantity
	}

	state.mu.Lock()
	// replace bank state
	state.bank = newBank

	// prune wallet holdings for stocks that no longer exist in bank
	for _, walletMap := range state.walletStocks {
		for stockName := range walletMap {
			if _, existsInBank := state.bank[stockName]; !existsInBank {
				delete(walletMap, stockName)
			}
		}
	}
	state.mu.Unlock()

	c.Status(http.StatusOK)
}

func getBankStocks(c *gin.Context, state *MarketState) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	out := make([]StockQty, 0, len(state.bank))
	for name, qty := range state.bank {
		out = append(out, StockQty{Name: name, Quantity: qty})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	c.JSON(http.StatusOK, BankResponse{Stocks: out})
}

func tradeOne(c *gin.Context, state *MarketState) {
	walletID := c.Param("wallet_id")
	stockName := c.Param("stock_name")

	var req TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Type != "buy" && req.Type != "sell" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'buy' or 'sell'"})
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	bankQty, stockExists := state.bank[stockName]
	if !stockExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
		return
	}

	// wallet autocreate on first successful call attempt
	if _, ok := state.walletStocks[walletID]; !ok {
		state.walletStocks[walletID] = map[string]int64{}
	}

	switch req.Type {
	case "buy":
		if bankQty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bank out of stock"})
			return
		}
		state.bank[stockName] = bankQty - 1
		state.walletStocks[walletID][stockName]++

	case "sell":
		wQty := state.walletStocks[walletID][stockName]
		if wQty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "wallet out of stock"})
			return
		}
		state.walletStocks[walletID][stockName] = wQty - 1
		state.bank[stockName] = bankQty + 1
	}

	state.audit = append(state.audit, AuditLogEntry{
		Type:      req.Type,
		WalletID:  walletID,
		StockName: stockName,
	})

	c.Status(http.StatusOK)
}

func getWallet(c *gin.Context, state *MarketState) {
	walletID := c.Param("wallet_id")

	state.mu.RLock()
	defer state.mu.RUnlock()

	walletMap, exists := state.walletStocks[walletID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	stocks := make([]StockQty, 0, len(walletMap))
	for name, qty := range walletMap {
		if qty > 0 {
			stocks = append(stocks, StockQty{Name: name, Quantity: qty})
		}
	}
	sort.Slice(stocks, func(i, j int) bool { return stocks[i].Name < stocks[j].Name })

	c.JSON(http.StatusOK, WalletResponse{ID: walletID, Stocks: stocks})
}

func getWalletStockQty(c *gin.Context, state *MarketState) {
	walletID := c.Param("wallet_id")
	stockName := c.Param("stock_name")

	state.mu.RLock()
	defer state.mu.RUnlock()

	if _, stockExists := state.bank[stockName]; !stockExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
		return
	}

	walletMap, walletExists := state.walletStocks[walletID]
	if !walletExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	qty := walletMap[stockName]
	c.String(http.StatusOK, "%d", qty)
}

func getAuditLog(c *gin.Context, state *MarketState) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	out := make([]AuditLogEntry, len(state.audit))
	copy(out, state.audit)

	c.JSON(http.StatusOK, AuditLogResponse{Log: out})
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
