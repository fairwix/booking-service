# Resource Booking Service

A Go service for booking shared resources. It exposes the same booking operations over **HTTP and gRPC**, keeps concurrent access to its in-memory state safe, and can run locally, with Docker Compose, or in a local Kubernetes cluster.

## What it does

- Books a meeting room or desk and returns their current availability.
- Shares one application service between the HTTP and gRPC transports.
- Maps validation and booking errors to appropriate HTTP status codes and gRPC codes.
- Exposes HTTP `/health` and the standard `grpc.health.v1.Health` service.
- Shuts down both servers gracefully on `SIGINT` or `SIGTERM`.
- Uses a multi-stage Docker build, Compose health checks, and a two-replica Kubernetes Deployment with separate liveness and readiness probes.

> [!IMPORTANT]
> Booking state is currently **in memory**. PostgreSQL is connected and pinged when `DATABASE_URL` is set, but bookings are not stored there yet. Each Kubernetes replica therefore has its own state. The two-replica setup demonstrates orchestration; it is not a consistent multi-replica booking system until persistence is added.

## Quick start

Requires Go 1.27.1. From the repository root:

```bash
go run ./cmd/service
```

The HTTP server listens on `8080` and gRPC on `9090` by default. Verify them with:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/resources
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

[`grpcurl`](https://github.com/fullstorydev/grpcurl) is needed only for the gRPC command.

### Docker Compose

```bash
docker compose up --build --wait
```

Compose starts PostgreSQL, waits for its `pg_isready` health check, then starts the service. The service verifies its database connection before opening the HTTP and gRPC listeners. Its own health check calls HTTP `/health`.

To change the exposed ports, set `HTTP_PORT` and `GRPC_PORT` before starting Compose. Run `docker compose down` to stop the stack. The credentials in [`docker-compose.yml`](docker-compose.yml) are for local development only.

### Local Kubernetes

Requires Docker, Minikube, and `kubectl`:

```bash
minikube start
./scripts/k8s-deploy.sh
./scripts/k8s-status.sh
kubectl port-forward service/booking-service 8080:8080 9090:9090
```

The deploy script builds `resource-booking-service:local`, loads it into Minikube, applies [`k8s/`](k8s/), and waits for two ready replicas. The `ClusterIP` Service exposes both transports inside the cluster. The image is local (`imagePullPolicy: Never`), so it must be loaded again for a fresh Minikube cluster.

Kubernetes checks HTTP `/health` for liveness and the native gRPC health protocol for readiness. The Minikube environment used for this project runs Kubernetes v1.37.0, which supports the native gRPC probe; no extra probe binary is needed. `requests` reserve capacity for scheduling, while `limits` cap CPU and memory use. Ports and the shutdown timeout are passed to the application through a ConfigMap. Numeric Kubernetes fields, including the gRPC probe port, must still match those values in the manifests.

## API

| Transport | Operation | Request | Result |
| --- | --- | --- | --- |
| HTTP | `GET /health` | — | `200 {"status":"ok"}` |
| HTTP | `GET /resources` | — | Resource IDs and `available`/`booked` statuses |
| HTTP | `POST /resources/book` | `{"id":1,"attendees":4}` | `204 No Content` on success |
| gRPC | `booking.BookingService/GetState` | Empty request | Resource states |
| gRPC | `booking.BookingService/Book` | ID and attendees | Empty response on success |
| gRPC | `grpc.health.v1.Health/Check` | Empty request | `SERVING` while ready |

Example booking:

```bash
curl -i -X POST http://localhost:8080/resources/book \
  -H 'Content-Type: application/json' \
  -d '{"id":1,"attendees":4}'
```

The initial resources are meeting room `1` (capacity 10) and desk `2`. A resource can be booked once until the process restarts. The [protobuf contract](proto/booking.proto) defines the gRPC API; server reflection is enabled.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_PORT` | `8080` | HTTP listener port |
| `GRPC_PORT` | `9090` | gRPC listener port |
| `SHUTDOWN_TIMEOUT` | `5s` | Maximum time for graceful shutdown; must be positive |
| `DATABASE_URL` | unset | Optional PostgreSQL connection URL; startup pings the database when set |

## Project layout

```text
cmd/service/              HTTP and gRPC server startup
internal/model/           Resource types and domain rules
internal/service/         Booking operations and synchronization
internal/transport/       HTTP and gRPC adapters
proto/                    Protocol Buffers contract and generated Go code
k8s/                      Deployment, Service, and ConfigMap
scripts/                  Local build, deploy, and orchestration demos
```

## Build and verification

```bash
go test ./...
go vet ./...
./scripts/docker-build.sh
kubectl apply --dry-run=client -f k8s/
```

`Dockerfile` builds a static binary with `CGO_ENABLED=0` in a Go builder stage and copies only that binary into Alpine. `Dockerfile.single` keeps the full Go image for comparison; `scripts/docker-build.sh` builds both variants and lists their sizes.

The demo scripts can show pod replacement and a rolling update in Minikube. For a rolling update, run `./scripts/prepare-demo.sh` followed by `./scripts/demo-rolling-update.sh`. The `v1` and `v2` tags in that demo illustrate deployment mechanics; they do not claim two different feature releases.

## Deployment report

The Docker and Kubernetes lab report includes the recorded image-size comparison, Compose checks, cluster configuration, and screenshots of Pod recovery and rolling update: [Lab 2 report](docs/lab2-report.md).

## Next step

Move booking state into PostgreSQL with transactional reservation checks so all replicas observe the same availability. The current database connection is an infrastructure starting point for that work.
