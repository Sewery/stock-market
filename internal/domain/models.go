package domain

type StockQty struct {
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type WalletResponse struct {
	ID     string     `json:"id"`
	Stocks []StockQty `json:"stocks"`
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

type AuditLogEntry struct {
	Type      string `json:"type"` // "buy" | "sell"
	WalletID  string `json:"wallet_id"`
	StockName string `json:"stock_name"`
}

type AuditLogResponse struct {
	Log []AuditLogEntry `json:"log"`
}
