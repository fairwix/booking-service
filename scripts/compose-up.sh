#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

docker compose up --build -d --wait
docker compose ps
curl -fsS -i "http://localhost:${HTTP_PORT:-8080}/health"

if command -v grpcurl >/dev/null 2>&1; then
  grpcurl -plaintext "localhost:${GRPC_PORT:-9090}" grpc.health.v1.Health/Check
else
  printf 'grpcurl is not installed; skipping the optional gRPC check.\n'
fi
