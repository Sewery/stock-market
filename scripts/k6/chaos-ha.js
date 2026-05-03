import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    steady_load: {
      executor: "shared-iterations",
      iterations: 10000,
      vus: 200,
      maxDuration: "30s",
      gracefulStop: "0s"
    },
    kill_app: {
      executor: "per-vu-iterations",
      vus: 1,
      iterations: 1,
      startTime: "2s",
      exec: "kill_app",
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

export function kill_app() {
  http.post(`${BASE_URL}/chaos`, null);

  let ok = false;
  for (let i = 0; i < 30; i++) {
    const r = http.get(`${BASE_URL}/stocks`);
    if (r.status === 200) { ok = true; break; }
    sleep(1);
  }
  check({ ok }, { "recovered": (v) => v.ok === true });
}

export default function () {
  const walletId = `w-${__VU}-${__ITER}`;
  const res = http.post(`${BASE_URL}/wallets/${walletId}/stocks/${STOCK}`,
    JSON.stringify({ type: "buy" }),
    { headers: { "Content-Type": "application/json" } }
  );
  check(res, { "buy ok or server error": (r) => r.status === 200 || r.status >= 500 });
  sleep(0.001);
}