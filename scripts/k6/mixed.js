import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    mixed_load: {
      executor: "constant-arrival-rate",
      rate: 200,       // 200 iters/s
      timeUnit: "1s",
      duration: "30s",
      preAllocatedVUs: 50,
      maxVUs: 200
    }
  }
};

const BASE_URL = __ENV.BASE_URL || "http://nginx:8080";
const STOCK = __ENV.STOCK || "stock1";

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

  // 70% reads, 30% writes
  const r = Math.random();

  if (r < 0.7) {
    const r1 = http.get(`${BASE_URL}/stocks`);
    const r2 = http.get(`${BASE_URL}/wallets/${walletId}`);
    check(r1, { "GET /stocks 200": (x) => x.status === 200 });
    check(r2, { "GET /wallets 200/404": (x) => x.status === 200 || x.status === 404 });
  } else {
    const type = r < 0.85 ? "buy" : "sell";
    const res = http.post(
      `${BASE_URL}/wallets/${walletId}/stocks/${STOCK}`,
      JSON.stringify({ type }),
      { headers: { "Content-Type": "application/json" } }
    );
    check(res, { "trade ok/400/404": (x) => [200, 400, 404].includes(x.status) });
  }

  sleep(0.001);
}