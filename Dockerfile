FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/avatars-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/avatars-worker ./cmd/worker

FROM migrate/migrate:v4.19.1 AS migrate
COPY migrations /migrations
ENTRYPOINT ["migrate"]

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates \
    && addgroup -g 1000 app \
    && adduser -D -u 1000 -G app app
WORKDIR /app
COPY --from=builder /out/avatars-api /app/avatars-api
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/app/avatars-api"]

FROM alpine:3.21 AS worker
RUN apk add --no-cache ca-certificates \
    && addgroup -g 1000 app \
    && adduser -D -u 1000 -G app app
WORKDIR /app
COPY --from=builder /out/avatars-worker /app/avatars-worker
USER 1000:1000
EXPOSE 9091
ENTRYPOINT ["/app/avatars-worker"]

FROM migrate AS avatars-migrate
