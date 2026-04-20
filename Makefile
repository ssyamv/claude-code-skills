BIN_DIR := dist
BINARY := $(BIN_DIR)/xfchat-bootstrapper

.PHONY: build test clean

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BINARY) ./cmd/xfchat-bootstrapper

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)
