#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

image=resource-booking-service:local
minikube status
docker build -t "$image" .
minikube image load --overwrite "$image"
kubectl apply -f k8s/
kubectl rollout restart deployment/booking-service
kubectl rollout status deployment/booking-service --timeout=120s
kubectl get deployment booking-service
kubectl get service booking-service
kubectl get pods -l app=booking-service
