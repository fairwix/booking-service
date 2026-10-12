#!/usr/bin/env bash
set -euo pipefail

echo "=== Node ==="
kubectl get nodes

echo
echo "=== Deployment ==="
kubectl get deployment booking-service

echo
echo "=== Pods ==="
kubectl get pods -l app=booking-service -o wide

echo
echo "=== Images and readiness ==="

kubectl get pods \
  -l app=booking-service \
  -o custom-columns='NAME:.metadata.name,IMAGE:.spec.containers[0].image,READY:.status.containerStatuses[0].ready'

echo
echo "=== Service ==="
kubectl get service booking-service

echo
echo "=== EndpointSlices ==="
kubectl get endpointslices \
  -l kubernetes.io/service-name=booking-service
