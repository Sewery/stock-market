package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"stock-market/internal/domain"
	"stock-market/internal/store"
)

type MemoryStore struct {
	mu sync.RWMutex

	bank         map[string]int64
	walletStocks map[string]map[string]int64
	audit        []domain.AuditLogEntry
}

func New() *MemoryStore {
	return &MemoryStore{
		bank:         map[string]int64{},
		walletStocks: map[string]map[string]int64{},
		audit:        make([]domain.AuditLogEntry, 0, 10000),
	}
}

func (s *MemoryStore) Close() {}

func (s *MemoryStore) SetBankStocks(_ context.Context, stocks []domain.StockQty) error {
	newBank := make(map[string]int64, len(stocks))
	for _, st := range stocks {
		if st.Name == "" {
			return fmt.Errorf("stock name is required")
		}
		if st.Quantity < 0 {
			return fmt.Errorf("stock quantity must be >= 0")
		}
		if _, exists := newBank[st.Name]; exists {
			return fmt.Errorf("duplicate stock name: %s", st.Name)
		}
		newBank[st.Name] = st.Quantity
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.bank = newBank

	for _, walletMap := range s.walletStocks {
		for stockName := range walletMap {
			if _, exists := s.bank[stockName]; !exists {
				delete(walletMap, stockName)
			}
		}
	}
	return nil
}

func (s *MemoryStore) GetBankStocks(_ context.Context) ([]domain.StockQty, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.StockQty, 0, len(s.bank))
	for name, qty := range s.bank {
		out = append(out, domain.StockQty{Name: name, Quantity: qty})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) TradeOne(_ context.Context, walletID, stockName, tradeType string) error {
	if tradeType != "buy" && tradeType != "sell" {
		return store.ErrInvalidTradeType
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bankQty, exists := s.bank[stockName]
	if !exists {
		return store.ErrStockNotFound
	}

	if _, ok := s.walletStocks[walletID]; !ok {
		s.walletStocks[walletID] = map[string]int64{}
	}

	switch tradeType {
	case "buy":
		if bankQty <= 0 {
			return store.ErrBankOutOfStock
		}
		s.bank[stockName] = bankQty - 1
		s.walletStocks[walletID][stockName]++
	case "sell":
		wQty := s.walletStocks[walletID][stockName]
		if wQty <= 0 {
			return store.ErrWalletOutOfStock
		}
		s.walletStocks[walletID][stockName] = wQty - 1
		s.bank[stockName] = bankQty + 1
	}

	s.audit = append(s.audit, domain.AuditLogEntry{
		Type:      tradeType,
		WalletID:  walletID,
		StockName: stockName,
	})
	return nil
}

func (s *MemoryStore) GetWallet(_ context.Context, walletID string) (domain.WalletResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	walletMap, ok := s.walletStocks[walletID]
	if !ok {
		return domain.WalletResponse{}, store.ErrWalletNotFound
	}

	stocks := make([]domain.StockQty, 0, len(walletMap))
	for name, qty := range walletMap {
		if qty > 0 {
			stocks = append(stocks, domain.StockQty{Name: name, Quantity: qty})
		}
	}
	sort.Slice(stocks, func(i, j int) bool { return stocks[i].Name < stocks[j].Name })

	return domain.WalletResponse{ID: walletID, Stocks: stocks}, nil
}

func (s *MemoryStore) GetWalletStockQty(_ context.Context, walletID, stockName string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.bank[stockName]; !exists {
		return 0, store.ErrStockNotFound
	}
	walletMap, ok := s.walletStocks[walletID]
	if !ok {
		return 0, store.ErrWalletNotFound
	}
	return walletMap[stockName], nil
}

func (s *MemoryStore) GetAuditLog(_ context.Context) ([]domain.AuditLogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.AuditLogEntry, len(s.audit))
	copy(out, s.audit)
	return out, nil
}
