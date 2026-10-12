FROM golang:1.27.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /service ./cmd/service


FROM alpine:3.22 AS runtime

COPY --from=builder /service /service

ENV HTTP_PORT=8080
ENV GRPC_PORT=9090

EXPOSE 8080
EXPOSE 9090

ENTRYPOINT ["/service"]