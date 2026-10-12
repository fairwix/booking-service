#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

docker build -t resource-booking-service:multi -f Dockerfile .
docker build -t resource-booking-service:single -f Dockerfile.single .
docker image ls resource-booking-service
