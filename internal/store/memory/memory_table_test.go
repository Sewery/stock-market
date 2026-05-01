package memory

import (
	"context"
	"errors"
	"testing"

	"stock-market/internal/domain"
	"stock-market/internal/store"
)

func TestMemoryStore_TradeOne_Errors_TableDriven(t *testing.T) {
	type tc struct {
		name      string
		setup     func(t *testing.T, s *MemoryStore)
		walletID  string
		stockName string
		tradeType string
		wantErr   error
	}

	tests := []tc{
		{
			name: "invalid trade type",
			setup: func(t *testing.T, s *MemoryStore) {
				t.Helper()
				_ = s.SetBankStocks(context.Background(), []domain.StockQty{{Name: "stock1", Quantity: 1}})
			},
			walletID:  "w1",
			stockName: "stock1",
			tradeType: "hold",
			wantErr:   store.ErrInvalidTradeType,
		},
		{
			name: "buy unknown stock",
			setup: func(t *testing.T, s *MemoryStore) {
				t.Helper()
				_ = s.SetBankStocks(context.Background(), []domain.StockQty{{Name: "stock1", Quantity: 1}})
			},
			walletID:  "w1",
			stockName: "nope",
			tradeType: "buy",
			wantErr:   store.ErrStockNotFound,
		},
		{
			name: "buy when bank empty",
			setup: func(t *testing.T, s *MemoryStore) {
				t.Helper()
				_ = s.SetBankStocks(context.Background(), []domain.StockQty{{Name: "stock1", Quantity: 0}})
			},
			walletID:  "w1",
			stockName: "stock1",
			tradeType: "buy",
			wantErr:   store.ErrBankOutOfStock,
		},
		{
			name: "sell when wallet empty",
			setup: func(t *testing.T, s *MemoryStore) {
				t.Helper()
				_ = s.SetBankStocks(context.Background(), []domain.StockQty{{Name: "stock1", Quantity: 1}})
				// wallet exists after first trade attempt, but still has 0 of this stock
				_ = s.TradeOne(context.Background(), "w1", "stock1", "buy")
				_ = s.TradeOne(context.Background(), "w1", "stock1", "sell")
			},
			walletID:  "w1",
			stockName: "stock1",
			tradeType: "sell",
			wantErr:   store.ErrWalletOutOfStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := New()
			defer s.Close()

			if tt.setup != nil {
				tt.setup(t, s)
			}

			err := s.TradeOne(ctx, tt.walletID, tt.stockName, tt.tradeType)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err=%v want=%v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryStore_Flow_HappyPath(t *testing.T) {
	ctx := context.Background()
	s := New()
	defer s.Close()

	walletID := "w1"
	stock := "stock1"

	if err := s.SetBankStocks(ctx, []domain.StockQty{{Name: stock, Quantity: 1}}); err != nil {
		t.Fatalf("seed bank: %v", err)
	}

	if _, err := s.GetWallet(ctx, walletID); !errors.Is(err, store.ErrWalletNotFound) {
		t.Fatalf("wallet should not exist: got %v", err)
	}

	if err := s.TradeOne(ctx, walletID, stock, "buy"); err != nil {
		t.Fatalf("buy: %v", err)
	}

	bank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("get bank: %v", err)
	}
	if len(bank) != 1 || bank[0].Name != stock || bank[0].Quantity != 0 {
		t.Fatalf("bank after buy: %#v", bank)
	}

	qty, err := s.GetWalletStockQty(ctx, walletID, stock)
	if err != nil {
		t.Fatalf("wallet qty: %v", err)
	}
	if qty != 1 {
		t.Fatalf("wallet qty after buy: got %d want %d", qty, 1)
	}

	if err := s.TradeOne(ctx, walletID, stock, "sell"); err != nil {
		t.Fatalf("sell: %v", err)
	}

	qty, err = s.GetWalletStockQty(ctx, walletID, stock)
	if err != nil {
		t.Fatalf("wallet qty after sell: %v", err)
	}
	if qty != 0 {
		t.Fatalf("wallet qty after sell: got %d want %d", qty, 0)
	}

	bank, err = s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("get bank after sell: %v", err)
	}
	if len(bank) != 1 || bank[0].Name != stock || bank[0].Quantity != 1 {
		t.Fatalf("bank after sell: %#v", bank)
	}

	log, err := s.GetAuditLog(ctx)
	if err != nil {
		t.Fatalf("get log: %v", err)
	}
	if len(log) != 2 {
		t.Fatalf("log len: got %d want %d", len(log), 2)
	}
	if log[0].Type != "buy" || log[1].Type != "sell" {
		t.Fatalf("log unexpected: %#v", log)
	}
}

func TestMemoryStore_SetBankStocks_Validation_TableDriven(t *testing.T) {
	type tc struct {
		name    string
		stocks  []domain.StockQty
		wantErr bool
	}

	tests := []tc{
		{"ok", []domain.StockQty{{Name: "stock1", Quantity: 0}}, false},
		{"empty name", []domain.StockQty{{Name: "", Quantity: 1}}, true},
		{"negative quantity", []domain.StockQty{{Name: "s", Quantity: -1}}, true},
		{"duplicate name", []domain.StockQty{{Name: "s", Quantity: 1}, {Name: "s", Quantity: 2}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			defer s.Close()

			err := s.SetBankStocks(context.Background(), tt.stocks)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}
