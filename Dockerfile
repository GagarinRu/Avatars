FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/avatars-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/avatars-worker ./cmd/worker

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/avatars-api /app/avatars-api
EXPOSE 8080
ENTRYPOINT ["/app/avatars-api"]

FROM alpine:3.21 AS worker
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/avatars-worker /app/avatars-worker
ENTRYPOINT ["/app/avatars-worker"]
