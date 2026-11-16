#!/usr/bin/env bash
set -euo pipefail

# Jynko End-to-End (Go-Rod) Test Runner for Local, CI/CD & Remote Cloud Run Environments
# Usage:
#   Local:  ./scripts/run-e2e.sh [OUT_DIR]
#   Remote: JYNKO_E2E_IAP_AUDIENCE="/projects/<NUM>/locations/<REGION>/services/jynko-plan-dev" \
#             ./scripts/run-e2e.sh [OUT_DIR] https://jynko-plan-dev-<hash>.a.run.app

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-${JYNKO_E2E_OUT_DIR:-/tmp/jynko-e2e-artifacts}}"
BASE_URL="${2:-${JYNKO_E2E_BASE_URL:-http://localhost:4202}}"
API_HEALTH_URL="${JYNKO_E2E_API_HEALTH_URL:-http://localhost:8082/healthz}"
BACKEND_PID=""
SERVER_PID=""

BACKEND_LOG="${TMPDIR:-/tmp}/jynko-backend-e2e-$$.log"
NG_LOG="${TMPDIR:-/tmp}/jynko-ng-e2e-$$.log"

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "==> Stopping background Angular dev server (PID ${SERVER_PID})..."
    pkill -P "${SERVER_PID}" 2>/dev/null || true
    kill "${SERVER_PID}" 2>/dev/null || true
  fi
  if [[ -n "${BACKEND_PID}" ]] && kill -0 "${BACKEND_PID}" 2>/dev/null; then
    echo "==> Stopping background Go backend server (PID ${BACKEND_PID})..."
    kill "${BACKEND_PID}" 2>/dev/null || true
  fi
  rm -f "${BACKEND_LOG}" "${NG_LOG}" 2>/dev/null || true
}
trap cleanup EXIT

IS_LOCAL_TARGET=0
if [[ "${BASE_URL}" =~ ^https?://(localhost|127\.0\.0\.1)(:|/|$) ]]; then
  IS_LOCAL_TARGET=1
fi

if [[ "${IS_LOCAL_TARGET}" -eq 1 ]]; then
  EXISTING_LISTEN_PIDS="$(lsof -tiTCP:8082 -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "${EXISTING_LISTEN_PIDS}" ]]; then
    echo "==> Restarting existing Go backend listener on :8082..."
    # shellcheck disable=SC2086
    kill ${EXISTING_LISTEN_PIDS} 2>/dev/null || true
    sleep 1
  fi

  echo "==> Building and starting Go backend server at ${API_HEALTH_URL}..."
  go -C "${REPO_ROOT}/backend" build -o /tmp/jynko-server-e2e ./cmd/server/main.go
  (cd "${REPO_ROOT}/backend" && ENV=dev SEED_ON_STARTUP=true exec /tmp/jynko-server-e2e) >"${BACKEND_LOG}" 2>&1 &
  BACKEND_PID=$!

  echo "==> Waiting for Go backend server at ${API_HEALTH_URL}..."
  BACKEND_READY=0
  for _ in $(seq 1 30); do
    if curl -fsS "${API_HEALTH_URL}" >/dev/null 2>&1; then
      BACKEND_READY=1
      break
    fi
    if ! kill -0 "${BACKEND_PID}" 2>/dev/null; then
      break
    fi
    sleep 1
  done

  if [[ "${BACKEND_READY}" -ne 1 ]]; then
    echo "ERROR: Backend failed to start at ${API_HEALTH_URL} within 30s. Logs:" >&2
    cat "${BACKEND_LOG}" >&2 || true
    exit 1
  fi

  if ! curl -fsS "${BASE_URL}" >/dev/null 2>&1; then
    echo "==> Frontend not detected at ${BASE_URL}; starting Angular dev server..."
    (cd "${REPO_ROOT}/frontend" && CI=true NG_CLI_ANALYTICS=false exec ./node_modules/.bin/ng serve --project jynko-frontend --host 127.0.0.1 --port 4202) >"${NG_LOG}" 2>&1 &
    SERVER_PID=$!

    echo "==> Waiting for Angular dev server at ${BASE_URL}..."
    READY=0
    for _ in $(seq 1 60); do
      if curl -fsS "${BASE_URL}" >/dev/null 2>&1; then
        READY=1
        break
      fi
      if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
        break
      fi
      sleep 1
    done

    if [[ "${READY}" -ne 1 ]]; then
      echo "ERROR: Frontend failed to start at ${BASE_URL} within 60s. Logs:" >&2
      cat "${NG_LOG}" >&2 || true
      exit 1
    fi
  fi
else
  echo "==> Remote target detected (${BASE_URL}); skipping local server startup."
  if [[ -z "${JYNKO_E2E_BEARER_TOKEN:-}" ]] && [[ -n "${JYNKO_E2E_IAP_AUDIENCE:-}" ]]; then
    echo "==> Minting OIDC identity token via gcloud for audience '${JYNKO_E2E_IAP_AUDIENCE}'..."
    JYNKO_E2E_BEARER_TOKEN="$(gcloud auth print-identity-token --audiences="${JYNKO_E2E_IAP_AUDIENCE}")"
    export JYNKO_E2E_BEARER_TOKEN
  fi
fi

echo "==> Running Jynko Go-Rod E2E Test Suite against ${BASE_URL} (Output: ${OUT_DIR})..."
JYNKO_E2E=1 \
JYNKO_E2E_BASE_URL="${BASE_URL}" \
JYNKO_E2E_OUT_DIR="${OUT_DIR}" \
go -C "${REPO_ROOT}/backend" test -v -count=1 ./internal/e2e/...
