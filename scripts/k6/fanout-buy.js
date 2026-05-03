import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    fanout_buy: {
      executor: "shared-iterations",
      iterations: 10000,
      vus: 200,
      maxDuration: "30s"
    }
  }
};

const BASE_URL = __ENV.BASE_URL || "http://nginx:8080";
const STOCK = __ENV.STOCK || "stock1";
const MAX_BUY = Number(__ENV.MAX_BUY || 200); // 1..MAX_BUY

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
  const walletId = `w-${__VU}-${__ITER}`;
  const qty = 1 + Math.floor(Math.random() * MAX_BUY);

  for (let i = 0; i < qty; i++) {
    const res = http.post(
      `${BASE_URL}/wallets/${walletId}/stocks/${STOCK}`,
      JSON.stringify({ type: "buy" }),
      { headers: { "Content-Type": "application/json" } }
    );
    check(res, { "buy ok/400/404": (r) => r.status === 200 || r.status === 400 || r.status === 404 });
  }

  sleep(0.001);
}