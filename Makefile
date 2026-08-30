.PHONY: test lint build cover

VERSION ?= 0.1.0
DATE ?= $(shell powershell -NoProfile -Command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ'")
COMMIT ?= $(shell git rev-parse --short HEAD 2>NUL || echo unknown)
LDFLAGS = -X main.buildVersion=$(VERSION) -X main.buildDate=$(DATE) -X main.buildCommit=$(COMMIT)

test:
	go test ./...

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...

cover:
	go test ./internal/... -coverprofile=coverage.out -covermode=atomic
	-go tool cover -func=coverage.out

build:
	go build -ldflags "$(LDFLAGS)" -o bin/avatars-api ./cmd/api
	go build -ldflags "$(LDFLAGS)" -o bin/avatars-worker ./cmd/worker
