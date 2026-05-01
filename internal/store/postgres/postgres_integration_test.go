package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"stock-market/internal/domain"
	"stock-market/internal/store"
	"stock-market/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgres_ConcurrentBuy_IsAtomic(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping postgres integration test")
	}

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
	defer s.Close()

	ctx := context.Background()
	if err := s.SetBankStocks(ctx, []domain.StockQty{{Name: "stock1", Quantity: 1}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	walletID := "w1"

	var wg sync.WaitGroup
	wg.Add(2)

	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			errs[i] = s.TradeOne(ctx, walletID, "stock1", "buy")
		}()
	}
	wg.Wait()

	successes := 0
	outOfStock := 0
	for _, e := range errs {
		if e == nil {
			successes++
			continue
		}
		if errors.Is(e, store.ErrBankOutOfStock) {
			outOfStock++
			continue
		}
		t.Fatalf("unexpected error: %v", e)
	}

	if successes != 1 || outOfStock != 1 {
		t.Fatalf("expected 1 success and 1 ErrBankOutOfStock, got successes=%d outOfStock=%d errs=%v", successes, outOfStock, errs)
	}

	bank, err := s.GetBankStocks(ctx)
	if err != nil {
		t.Fatalf("GetBankStocks: %v", err)
	}
	if len(bank) != 1 || bank[0].Name != "stock1" || bank[0].Quantity != 0 {
		t.Fatalf("bank unexpected: %#v", bank)
	}

	qty, err := s.GetWalletStockQty(ctx, walletID, "stock1")
	if err != nil {
		t.Fatalf("GetWalletStockQty: %v", err)
	}
	if qty != 1 {
		t.Fatalf("wallet qty got %d want %d", qty, 1)
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
