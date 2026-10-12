#!/usr/bin/env bash
set -u

health_url=${HEALTH_URL:-http://localhost:8080/health}
printf 'Checking %s once per second. Press Ctrl+C to stop.\n' "$health_url"
printf 'A kubectl port-forward tunnel can disconnect when its selected Pod is replaced.\n'

while true; do
    printf '%s ' "$(date '+%H:%M:%S')"

    curl \
      -s \
      -o /dev/null \
      -w 'HTTP %{http_code}\n' \
      "$health_url"

    sleep 1
done
