#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PORT=18080
LOG_FILE="${TMPDIR:-/tmp}/quant-hybrid-control-plane.log"

"$ROOT/bin/control-plane" --engine "$ROOT/bin/execution_core" --addr "127.0.0.1:$PORT" >"$LOG_FILE" 2>&1 &
SERVER_PID=$!
cleanup() {
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 0.1
done
[ "$ready" -eq 1 ] || { printf '%s\n' "control plane did not become ready"; exit 1; }

health=$(curl -fsS "http://127.0.0.1:$PORT/health")
case "$health" in *'"status":"ok"'*) ;; *) printf '%s\n' "health check failed: $health"; exit 1 ;; esac

accepted=$(curl -fsS -X POST "http://127.0.0.1:$PORT/orders" \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"smoke-1","symbol":"DEMO","side":"BUY","quantity":10,"limit_price":100.00}')
case "$accepted" in *'"status":"accepted_by_control_plane"'*) ;; *) printf '%s\n' "order submission failed: $accepted"; exit 1 ;; esac

invalid_status=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:$PORT/orders" \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"DEMO","side":"BUY","quantity":101,"limit_price":100.00}')
[ "$invalid_status" = "400" ] || { printf '%s\n' "risk rejection failed: HTTP $invalid_status"; exit 1; }

snapshot=''
position_ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  snapshot=$(curl -fsS "http://127.0.0.1:$PORT/snapshot")
  case "$snapshot" in *'"DEMO":10'*) position_ready=1; break ;; esac
  sleep 0.1
done
[ "$position_ready" -eq 1 ] || { printf '%s\n' "position snapshot failed: $snapshot"; exit 1; }

printf '%s\n' "smoke: health=ok order=accepted risk=blocked position=10"
