package store

import "errors"

var (
	ErrStockNotFound    = errors.New("stock not found")
	ErrBankOutOfStock   = errors.New("bank out of stock")
	ErrWalletOutOfStock = errors.New("wallet out of stock")
	ErrWalletNotFound   = errors.New("wallet not found")
	ErrInvalidTradeType = errors.New("invalid trade type")
)
