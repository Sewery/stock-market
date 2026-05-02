import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    hot_wallet: {
      executor: "constant-arrival-rate",
      rate: 400,
      timeUnit: "1s",
      duration: "25s",
      preAllocatedVUs: 50,
      maxVUs: 200
    }
  }
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const STOCK = __ENV.STOCK || "stock1";
const WALLET = __ENV.WALLET || "w-hot";

export function setup() {
  http.post(`${BASE_URL}/stocks`, JSON.stringify({ stocks: [{ name: STOCK, quantity: 100000 }] }), {
    headers: { "Content-Type": "application/json" }
  });
}

export function teardown() {
  http.post(`${BASE_URL}/stocks`, JSON.stringify({ stocks: [] }), {
    headers: { "Content-Type": "application/json" }
  });
}

export default function () {
  const buy = http.post(`${BASE_URL}/wallets/${WALLET}/stocks/${STOCK}`,
    JSON.stringify({ type: "buy" }),
    { headers: { "Content-Type": "application/json" } }
  );
  const sell = http.post(`${BASE_URL}/wallets/${WALLET}/stocks/${STOCK}`,
    JSON.stringify({ type: "sell" }),
    { headers: { "Content-Type": "application/json" } }
  );

  check(buy, { "buy ok/400": (r) => r.status === 200 || r.status === 400 });
  check(sell, { "sell ok/400": (r) => r.status === 200 || r.status === 400 });
  sleep(0.001);
}