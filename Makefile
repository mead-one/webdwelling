# vim: set noexpandtab:
BINARY_NAME=webdwelling
BINARY_DIR=bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)

# Load secrets from .env if it exists (see .env.default)
-include .env
export JWT_SECRET

all: build_ts test build

build_ts:
	npx tsc

test:
	go test -v cmd/server/main.go
	go vet ./...

build:
	go build -o $(BINARY_PATH) cmd/server/main.go

run:
	npx tsc
	go run cmd/server/main.go

clean:
	go clean
	rm -rf $(BINARY_DIR)

registeruser:
	go run cmd/registeruser/main.go $(ARGS)
