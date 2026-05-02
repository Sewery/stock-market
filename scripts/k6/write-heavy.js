import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    write_heavy: {
      executor: "constant-arrival-rate",
      rate: 1000,      // 1000 ops/s
      timeUnit: "1s",
      duration: "60s",
      preAllocatedVUs: 200,
      maxVUs: 1000
    }
  }
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const STOCK = __ENV.STOCK || "stock1";

export function setup() {
  const payload = JSON.stringify({ stocks: [{ name: STOCK, quantity: 1000000 }] });
  const res = http.post(`${BASE_URL}/stocks`, payload, { headers: { "Content-Type": "application/json" } });
  check(res, { "seed ok": (r) => r.status === 200 });
}

export default function () {
  const walletId = `w-${__VU}-${__ITER}`;
  const url = `${BASE_URL}/wallets/${walletId}/stocks/${STOCK}`;
  const res = http.post(url, JSON.stringify({ type: "buy" }), {
    headers: { "Content-Type": "application/json" }
  });
  check(res, { "buy ok or out-of-stock": (r) => r.status === 200 || r.status === 400 });
  sleep(0.001);
}