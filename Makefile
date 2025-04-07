# vim: set noexpandtab:
BINARY_NAME=webdwelling
BINARY_DIR=bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)

all: build_ts test build

build_ts:
	npx tsc

test:
	go test -v cmd/server/main.go

build:
	go build -o $(BINARY_PATH) cmd/server/main.go

run:
	npx tsc
	go run cmd/server/main.go

clean:
	go clean
	rm -rf $(BINARY_DIR)
