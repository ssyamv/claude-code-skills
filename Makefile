BIN_DIR := dist
BINARY := $(BIN_DIR)/xfchat-bootstrapper

.PHONY: build test clean release

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BINARY) ./cmd/xfchat-bootstrapper

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)

release:
	bash ./scripts/build-release.sh
	bash ./scripts/build-release-test.sh
