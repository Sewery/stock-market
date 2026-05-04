param(
  [string]$BaseUrl = "http://localhost:8080"
)

$WalletId = "w-$([guid]::NewGuid().ToString('N').Substring(0,8))"
$Stock = "stock1"

function Call-Api($Method, $Path, $Want, $Data = $null) {
  $Url = "$BaseUrl$Path"

  if ($Data) {
    $resp = Invoke-WebRequest -Method $Method -Uri $Url -ContentType "application/json" -Body $Data -UseBasicParsing -SkipHttpErrorCheck
  } else {
    $resp = Invoke-WebRequest -Method $Method -Uri $Url -UseBasicParsing -SkipHttpErrorCheck
  }

  $code = $resp.StatusCode
  $body = $resp.Content

  if ($code -ne $Want) {
    Write-Host "FAIL $Method $Path"
    Write-Host "  want: $Want"
    Write-Host "  got:  $code"
    Write-Host "  body: $body"
    exit 1
  }

  return $body
}

function Need-Contains($Body, $Needle, $Msg) {
  if (-not $Body.Contains($Needle)) {
    Write-Host "FAIL $Msg"
    Write-Host "  body: $Body"
    exit 1
  }
}

Write-Host "BASE_URL=$BaseUrl wallet=$WalletId stock=$Stock"

Write-Host "Seed bank"
Call-Api POST "/stocks" 200 "{`"stocks`":[{`"name`":`"$Stock`",`"quantity`":1}]}"

Write-Host "Wallet should not exist yet"
Call-Api GET "/wallets/$WalletId" 404 | Out-Null

Write-Host "Buy unknown stock should be 404"
Call-Api POST "/wallets/$WalletId/stocks/nope" 404 "{`"type`":`"buy`"}" | Out-Null

Write-Host "Buy stock (creates wallet)"
Call-Api POST "/wallets/$WalletId/stocks/$Stock" 200 "{`"type`":`"buy`"}" | Out-Null

Write-Host "Bank quantity should be 0"
$body = Call-Api GET "/stocks" 200
Need-Contains $body "`"name`":`"$Stock`"" "bank response missing stock"
Need-Contains $body "`"quantity`":0" "bank quantity is not 0"

Write-Host "Wallet stock qty should be 1"
$body = Call-Api GET "/wallets/$WalletId/stocks/$Stock" 200
if ($body -ne "1") { Write-Host "FAIL wallet qty expected 1, got '$body'"; exit 1 }

Write-Host "Wallet should contain stock qty 1"
$body = Call-Api GET "/wallets/$WalletId" 200
Need-Contains $body "`"name`":`"$Stock`"" "wallet missing stock"
Need-Contains $body "`"quantity`":1" "wallet quantity is not 1"

Write-Host "Buying again should fail (bank empty)"
Call-Api POST "/wallets/$WalletId/stocks/$Stock" 400 "{`"type`":`"buy`"}" | Out-Null

Write-Host "Sell stock"
Call-Api POST "/wallets/$WalletId/stocks/$Stock" 200 "{`"type`":`"sell`"}" | Out-Null

Write-Host "Wallet stock qty should be 0 after sell"
$body = Call-Api GET "/wallets/$WalletId/stocks/$Stock" 200
if ($body -ne "0") { Write-Host "FAIL wallet qty expected 0, got '$body'"; exit 1 }

Write-Host "Bank quantity should be back to 1"
$body = Call-Api GET "/stocks" 200
Need-Contains $body "`"name`":`"$Stock`"" "bank response missing stock"
Need-Contains $body "`"quantity`":1" "bank quantity is not 1"

Write-Host "Selling again should fail (wallet empty)"
Call-Api POST "/wallets/$WalletId/stocks/$Stock" 400 "{`"type`":`"sell`"}" | Out-Null

Write-Host "Audit log should include buy/sell for this wallet"
$body = Call-Api GET "/log" 200
Need-Contains $body "`"wallet_id`":`"$WalletId`"" "log missing wallet entry"
Need-Contains $body "`"type`":`"buy`"" "log missing buy entry"
Need-Contains $body "`"type`":`"sell`"" "log missing sell entry"

Write-Host "OK"