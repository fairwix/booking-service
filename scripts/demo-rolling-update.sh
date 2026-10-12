#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

image=resource-booking-service:v2
docker build -t "$image" .
minikube image load --overwrite "$image"
kubectl set image deployment/booking-service booking-service="$image"
kubectl rollout status deployment/booking-service --timeout=120s
kubectl get pods -l app=booking-service \
  -o custom-columns='NAME:.metadata.name,IMAGE:.spec.containers[0].image,READY:.status.containerStatuses[0].ready'
