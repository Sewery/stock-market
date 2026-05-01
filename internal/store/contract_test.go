package store_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"stock-market/internal/domain"
	"stock-market/internal/store"
	"stock-market/internal/store/memory"
	"stock-market/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStoreContract_Memory(t *testing.T) {
	runStoreContract(t, func(t *testing.T) store.Store {
		t.Helper()
		s := memory.New()
		t.Cleanup(s.Close)
		return s
	})
}

func TestStoreContract_Postgres(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping postgres contract tests")
	}

	runStoreContract(t, func(t *testing.T) store.Store {
		t.Helper()

		root := repoRoot(t)
		migrationsDir := filepath.Join(root, "migrations")

		resetPostgres(t, dbURL, migrationsDir)

		s, err := postgres.New(context.Background(), postgres.Config{
			DatabaseURL:   dbURL,
			MigrationsDir: migrationsDir,
		})
		if err != nil {
			t.Fatalf("postgres.New: %v", err)
		}
		t.Cleanup(s.Close)
		return s
	})
}

func runStoreContract(t *testing.T, newStore func(t *testing.T) store.Store) {
	t.Helper()

	ctx := context.Background()
	s := newStore(t)

	stock := "stock1"
	wallet := "w1"

	// seed bank
	if err := s.SetBankStocks(ctx, []domain.StockQty{{Name: stock, Quantity: 1}}); err != nil {
		t.Fatalf("SetBankStocks: %v", err)
	}

	// wallet not found
	if _, err := s.GetWallet(ctx, wallet); !errors.Is(err, store.ErrWalletNotFound) {
		t.Fatalf("GetWallet should be ErrWalletNotFound, got %v", err)
	}

	// buy unknown stock
	if err := s.TradeOne(ctx, wallet, "nope", "buy"); !errors.Is(err, store.ErrStockNotFound) {
		t.Fatalf("buy unknown should be ErrStockNotFound, got %v", err)
	}

	// buy ok
	if err := s.TradeOne(ctx, wallet, stock, "buy"); err != nil {
		t.Fatalf("buy: %v", err)
	}

	// bank qty should be 0
	bank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks: %v", err)
	}
	if len(bank) != 1 || bank[0].Name != stock || bank[0].Quantity != 0 {
		t.Fatalf("bank unexpected: %#v", bank)
	}

	// wallet qty should be 1
	qty, err := s.GetWalletStockQty(ctx, wallet, stock)
	if err != nil {
		t.Fatalf("GetWalletStockQty: %v", err)
	}
	if qty != 1 {
		t.Fatalf("wallet qty got %d want %d", qty, 1)
	}

	// buy again -> bank out of stock
	if err := s.TradeOne(ctx, wallet, stock, "buy"); !errors.Is(err, store.ErrBankOutOfStock) {
		t.Fatalf("buy when empty should be ErrBankOutOfStock, got %v", err)
	}

	// sell ok
	if err := s.TradeOne(ctx, wallet, stock, "sell"); err != nil {
		t.Fatalf("sell: %v", err)
	}

	// wallet qty should be 0
	qty, err = s.GetWalletStockQty(ctx, wallet, stock)
	if err != nil {
		t.Fatalf("GetWalletStockQty after sell: %v", err)
	}
	if qty != 0 {
		t.Fatalf("wallet qty got %d want %d", qty, 0)
	}

	// bank qty should be 1
	bank, err = s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks after sell: %v", err)
	}
	if len(bank) != 1 || bank[0].Name != stock || bank[0].Quantity != 1 {
		t.Fatalf("bank unexpected: %#v", bank)
	}

	// log should have 2 entries: buy, sell
	log, err := s.GetAuditLog(ctx)
	if err != nil {
		t.Fatalf("GetAuditLog: %v", err)
	}
	if len(log) != 2 {
		t.Fatalf("log len got %d want %d (%#v)", len(log), 2, log)
	}
	if log[0].Type != "buy" || log[1].Type != "sell" {
		t.Fatalf("log unexpected: %#v", log)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not find repo root (go.mod) from wd")
	return ""
}

func resetPostgres(t *testing.T, dbURL, migrationsDir string) {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	b, err := os.ReadFile(filepath.Join(migrationsDir, "001_init.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(b)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	if _, err := pool.Exec(ctx, `
        TRUNCATE TABLE audit_log, wallet_stocks, wallets, stocks
        RESTART IDENTITY CASCADE;
    `); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
