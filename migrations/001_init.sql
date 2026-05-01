CREATE TABLE IF NOT EXISTS wallets (
  id TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS stocks (
  name TEXT PRIMARY KEY,
  quantity BIGINT NOT NULL CHECK (quantity >= 0)
);

CREATE TABLE IF NOT EXISTS wallet_stocks (
  wallet_id  TEXT NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  stock_name TEXT NOT NULL REFERENCES stocks(name) ON DELETE CASCADE,
  quantity   BIGINT NOT NULL CHECK (quantity >= 0),
  PRIMARY KEY (wallet_id, stock_name)
);

CREATE TABLE IF NOT EXISTS audit_log (
  seq        BIGSERIAL PRIMARY KEY,
  type       TEXT NOT NULL CHECK (type IN ('buy','sell')),
  wallet_id  TEXT NOT NULL,
  stock_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
