package store

import (
	"context"

	"stock-market/internal/domain"
)

type Store interface {
	SetBankStocks(ctx context.Context, stocks []domain.StockQty) error
	GetBankStocks(ctx context.Context) ([]domain.StockQty, error)

	TradeOne(ctx context.Context, walletID, stockName, tradeType string) error
	GetWallet(ctx context.Context, walletID string) (domain.WalletResponse, error)
	GetWalletStockQty(ctx context.Context, walletID, stockName string) (int64, error)

	GetAuditLog(ctx context.Context) ([]domain.AuditLogEntry, error)

	Close()
}
