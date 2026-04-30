#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
WALLET_ID="w-$RANDOM$RANDOM"
STOCK="stock1"

BODY=""
CODE=""

call() {
  local method="$1"
  local path="$2"
  local want="$3"
  local data="${4:-}"

  local url="${BASE_URL}${path}"
  local resp

  if [[ -n "$data" ]]; then
    resp="$(curl -sS -X "$method" \
      -H "Content-Type: application/json" \
      -d "$data" \
      -w $'\n%{http_code}' \
      "$url")"
  else
    resp="$(curl -sS -X "$method" \
      -w $'\n%{http_code}' \
      "$url")"
  fi

  CODE="$(printf '%s\n' "$resp" | tail -n1)"
  BODY="$(printf '%s\n' "$resp" | sed '$d')"

  if [[ "$CODE" != "$want" ]]; then
    echo "FAIL $method $path"
    echo "  want: $want"
    echo "  got:  $CODE"
    echo "  body: $BODY"
    exit 1
  fi
}

need() {
  local needle="$1"
  local msg="$2"
  echo "$BODY" | grep -q "$needle" || { echo "FAIL $msg"; echo "  body: $BODY"; exit 1; }
}

echo "BASE_URL=$BASE_URL wallet=$WALLET_ID stock=$STOCK"

echo "Seed bank"
call POST /stocks 200 "{\"stocks\":[{\"name\":\"$STOCK\",\"quantity\":1}]}"

echo "Wallet should not exist yet"
call GET "/wallets/$WALLET_ID" 404

echo "Buy unknown stock should be 404"
call POST "/wallets/$WALLET_ID/stocks/nope" 404 '{"type":"buy"}'

echo "Buy stock (creates wallet)"
call POST "/wallets/$WALLET_ID/stocks/$STOCK" 200 '{"type":"buy"}'

echo "Bank quantity should be 0"
call GET /stocks 200
need "\"name\":\"$STOCK\"" "bank response missing stock"
need "\"quantity\":0" "bank quantity is not 0"

echo "Wallet stock qty should be 1"
call GET "/wallets/$WALLET_ID/stocks/$STOCK" 200
[[ "$BODY" == "1" ]] || { echo "FAIL wallet qty expected 1, got '$BODY'"; exit 1; }

echo "Wallet should contain $STOCK qty 1"
call GET "/wallets/$WALLET_ID" 200
need "\"name\":\"$STOCK\"" "wallet missing stock"
need "\"quantity\":1" "wallet quantity is not 1"

echo "Buying again should fail (bank empty)"
call POST "/wallets/$WALLET_ID/stocks/$STOCK" 400 '{"type":"buy"}'

echo "Sell stock"
call POST "/wallets/$WALLET_ID/stocks/$STOCK" 200 '{"type":"sell"}'

echo "Wallet stock qty should be 0 after sell"
call GET "/wallets/$WALLET_ID/stocks/$STOCK" 200
[[ "$BODY" == "0" ]] || { echo "FAIL wallet qty expected 0, got '$BODY'"; exit 1; }

echo "Bank quantity should be back to 1"
call GET /stocks 200
need "\"name\":\"$STOCK\"" "bank response missing stock"
need "\"quantity\":1" "bank quantity is not 1"

echo "Selling again should fail (wallet empty)"
call POST "/wallets/$WALLET_ID/stocks/$STOCK" 400 '{"type":"sell"}'

echo "Audit log should include buy/sell for this wallet"
call GET /log 200
need "\"wallet_id\":\"$WALLET_ID\"" "log missing wallet entry"
need "\"type\":\"buy\"" "log missing buy entry"
need "\"type\":\"sell\"" "log missing sell entry"

echo "OK"