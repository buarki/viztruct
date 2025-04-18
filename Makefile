# viztruct - go struct memory layout visualizer

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOFMT=$(GOCMD) fmt
GOTEST=$(GOCMD) test --race
GOOS=js
GOARCH=wasm

CLI_NAME=viztruct
WASM_BINARY_NAME=main.wasm
OUTPUT_DIR=static
WASM_DIR=cmd/server
CLI_DIR=cmd/cli
WASM_EXEC_PATH=/usr/local/go/lib/wasm/wasm_exec.js

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.binVersion=$(VERSION)

build-wasm:
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -o $(OUTPUT_DIR)/$(WASM_BINARY_NAME) ./$(WASM_DIR)

build-cli:
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(CLI_NAME) ./$(CLI_DIR)

clean:
	$(GOCLEAN)
	rm -f $(OUTPUT_DIR)/$(WASM_BINARY_NAME)
	rm -rf bin/

wasm-exec:
	cp $(WASM_EXEC_PATH) ./static

fmt:
	$(GOFMT) ./...

test: fmt
	$(GOTEST) ./structi/... ./svg/...

serve:
	npx http-server ./static --cors

all: clean build-wasm build-cli

.PHONY: build-wasm clean fmt test serve all
