#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

kubectl get pods -l app=booking-service
pod=$(kubectl get pods -l app=booking-service -o jsonpath='{.items[0].metadata.name}')
printf 'Deleting %s\n' "$pod"
kubectl delete pod "$pod" --wait=true
kubectl wait --for=jsonpath='{.status.readyReplicas}'=2 deployment/booking-service --timeout=120s
kubectl get pods -l app=booking-service
