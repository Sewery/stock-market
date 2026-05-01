package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"stock-market/internal/domain"
	"stock-market/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

type Config struct {
	DatabaseURL   string
	MigrationsDir string // default: migrations
}

func New(ctx context.Context, cfg Config) (*PostgresStore, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	s := &PostgresStore{pool: pool}
	if err := s.Migrate(ctx, cfg.MigrationsDir); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

func (s *PostgresStore) Migrate(ctx context.Context, migrationsDir string) error {
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	path := filepath.Join(migrationsDir, "001_init.sql")
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", path, err)
	}
	if _, err := s.pool.Exec(ctx, string(b)); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	return nil
}

func (s *PostgresStore) SetBankStocks(ctx context.Context, stocks []domain.StockQty) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Blocking whole table here to prevent concurrent SetBankStocks or TradeOne calls
		if _, err := tx.Exec(ctx, `LOCK TABLE stocks IN EXCLUSIVE MODE`); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `DELETE FROM stocks`); err != nil {
			return err
		}

		for _, st := range stocks {
			if st.Name == "" {
				return fmt.Errorf("stock name is required")
			}
			if st.Quantity < 0 {
				return fmt.Errorf("stock quantity must be >= 0")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO stocks(name, quantity) VALUES ($1,$2)`, st.Name, st.Quantity); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PostgresStore) GetBankStocks(ctx context.Context) ([]domain.StockQty, error) {
	rows, err := s.pool.Query(ctx, `SELECT name, quantity FROM stocks ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.StockQty, 0)
	for rows.Next() {
		var st domain.StockQty
		if err := rows.Scan(&st.Name, &st.Quantity); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *PostgresStore) TradeOne(ctx context.Context, walletID, stockName, tradeType string) error {
	if tradeType != "buy" && tradeType != "sell" {
		return store.ErrInvalidTradeType
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var stockExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stocks WHERE name=$1)`, stockName).Scan(&stockExists); err != nil {
			return err
		}
		if !stockExists {
			return store.ErrStockNotFound
		}

		if _, err := tx.Exec(ctx, `INSERT INTO wallets(id) VALUES ($1) ON CONFLICT DO NOTHING`, walletID); err != nil {
			return err
		}

		switch tradeType {
		case "buy":
			cmd, err := tx.Exec(ctx, `UPDATE stocks SET quantity = quantity - 1 WHERE name=$1 AND quantity > 0`, stockName)
			if err != nil {
				return err
			}
			if cmd.RowsAffected() == 0 {
				return store.ErrBankOutOfStock
			}

			if _, err := tx.Exec(ctx, `INSERT INTO wallet_stocks(wallet_id, stock_name, quantity)
                VALUES ($1,$2,1)
                ON CONFLICT (wallet_id, stock_name) DO UPDATE SET quantity = wallet_stocks.quantity + 1`, walletID, stockName); err != nil {
				return err
			}

		case "sell":
			cmd, err := tx.Exec(ctx, `UPDATE wallet_stocks SET quantity = quantity - 1
                WHERE wallet_id=$1 AND stock_name=$2 AND quantity > 0`, walletID, stockName)
			if err != nil {
				return err
			}
			if cmd.RowsAffected() == 0 {
				return store.ErrWalletOutOfStock
			}

			if _, err := tx.Exec(ctx, `UPDATE stocks SET quantity = quantity + 1 WHERE name=$1`, stockName); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM wallet_stocks WHERE wallet_id=$1 AND stock_name=$2 AND quantity=0`, walletID, stockName); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, `INSERT INTO audit_log(type, wallet_id, stock_name) VALUES ($1,$2,$3)`, tradeType, walletID, stockName); err != nil {
			return err
		}
		return nil
	})
}

func (s *PostgresStore) GetWallet(ctx context.Context, walletID string) (domain.WalletResponse, error) {
	var walletExists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wallets WHERE id=$1)`, walletID).Scan(&walletExists); err != nil {
		return domain.WalletResponse{}, err
	}
	if !walletExists {
		return domain.WalletResponse{}, store.ErrWalletNotFound
	}

	rows, err := s.pool.Query(ctx, `SELECT stock_name, quantity FROM wallet_stocks WHERE wallet_id=$1 AND quantity > 0 ORDER BY stock_name`, walletID)
	if err != nil {
		return domain.WalletResponse{}, err
	}
	defer rows.Close()

	resp := domain.WalletResponse{ID: walletID}
	for rows.Next() {
		var name string
		var qty int64
		if err := rows.Scan(&name, &qty); err != nil {
			return domain.WalletResponse{}, err
		}
		resp.Stocks = append(resp.Stocks, domain.StockQty{Name: name, Quantity: qty})
	}
	if err := rows.Err(); err != nil {
		return domain.WalletResponse{}, err
	}
	return resp, nil
}

func (s *PostgresStore) GetWalletStockQty(ctx context.Context, walletID, stockName string) (int64, error) {
	var stockExists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stocks WHERE name=$1)`, stockName).Scan(&stockExists); err != nil {
		return 0, err
	}
	if !stockExists {
		return 0, store.ErrStockNotFound
	}

	var walletExists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wallets WHERE id=$1)`, walletID).Scan(&walletExists); err != nil {
		return 0, err
	}
	if !walletExists {
		return 0, store.ErrWalletNotFound
	}

	var qty int64
	err := s.pool.QueryRow(ctx, `SELECT quantity FROM wallet_stocks WHERE wallet_id=$1 AND stock_name=$2`, walletID, stockName).Scan(&qty)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return qty, nil
}

func (s *PostgresStore) GetAuditLog(ctx context.Context) ([]domain.AuditLogEntry, error) {
	rows, err := s.pool.Query(ctx, `SELECT type, wallet_id, stock_name FROM audit_log ORDER BY seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.AuditLogEntry, 0)
	for rows.Next() {
		var e domain.AuditLogEntry
		if err := rows.Scan(&e.Type, &e.WalletID, &e.StockName); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
