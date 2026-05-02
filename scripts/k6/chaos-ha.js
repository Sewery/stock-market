import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    steady_load: {
      executor: "constant-arrival-rate",
      rate: 500,
      timeUnit: "1s",
      duration: "90s",
      preAllocatedVUs: 100,
      maxVUs: 600
    },
    kill_app: {
      executor: "per-vu-iterations",
      vus: 1,
      iterations: 1,
      startTime: "30s"
    }
  }
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const STOCK = __ENV.STOCK || "stock1";

export function setup() {
  http.post(`${BASE_URL}/stocks`, JSON.stringify({ stocks: [{ name: STOCK, quantity: 100000 }] }), {
    headers: { "Content-Type": "application/json" }
  });
}

export function kill_app() {
  // wywołuje /chaos, które ubija proces
  http.post(`${BASE_URL}/chaos`, null);
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