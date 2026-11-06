# vim: set noexpandtab:
BINARY_NAME=webdwelling
BINARY_DIR=bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)

# Load secrets from .env if it exists (see .env.default)
-include .env
export JWT_SECRET

.PHONY: all build_ts test build run clean registeruser

all: build_ts test build

build_ts:
	npx tsc

test:
	go test ./...
	go vet ./...
	staticcheck ./...

build: build_ts
	go build -o $(BINARY_PATH) ./cmd/server

run: build_ts
	go run ./cmd/server

clean:
	go clean
	rm -rf $(BINARY_DIR)

registeruser:
	go run ./cmd/registeruser $(ARGS)
