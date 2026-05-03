import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    read_heavy: {
      executor: "shared-iterations",
      iterations: 10000,
      vus: 200,
      maxDuration: "20s"
    }
  }
};

const BASE_URL = __ENV.BASE_URL || "http://nginx:8080";
const WALLET = __ENV.WALLET || "w-read";
const STOCK = __ENV.STOCK || "stock1";

export function setup() {
  // seed minimal data
  http.post(`${BASE_URL}/stocks`, JSON.stringify({ stocks: [{ name: STOCK, quantity: 10 }] }), {
    headers: { "Content-Type": "application/json" }
  });
  http.post(`${BASE_URL}/wallets/${WALLET}/stocks/${STOCK}`, JSON.stringify({ type: "buy" }), {
    headers: { "Content-Type": "application/json" }
  });
}

export function teardown() {
  http.post(`${BASE_URL}/stocks`, JSON.stringify({ stocks: [] }), {
    headers: { "Content-Type": "application/json" }
  });
}

export default function () {
  const r1 = http.get(`${BASE_URL}/stocks`);
  const r2 = http.get(`${BASE_URL}/wallets/${WALLET}`);
  const r3 = http.get(`${BASE_URL}/wallets/${WALLET}/stocks/${STOCK}`);

  check(r1, { "GET /stocks": (r) => r.status === 200 });
  check(r2, { "GET /wallets": (r) => r.status === 200 });
  check(r3, { "GET /wallets/:id/stocks/:stock": (r) => r.status === 200 });

  sleep(0.001);
}