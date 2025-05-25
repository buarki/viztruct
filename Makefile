CLI_NAME=viztruct
WASM_BINARY_NAME=main.wasm
OUTPUT_DIR=static
WASM_DIR=cmd/server
CLI_DIR=cmd/viztruct
GO_INSTALL_PATH=$(shell which go)
WASM_EXEC_PATH=$(GO_INSTALL_PATH)/lib/wasm/wasm_exec.js
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.binVersion=$(VERSION)

default: build

build-wasm:
	GOOS=js GOARCH=wasm go build -o $(OUTPUT_DIR)/$(WASM_BINARY_NAME) ./$(WASM_DIR)

build-cli:
	go build -ldflags "$(LDFLAGS)" -o $(CLI_NAME) ./$(CLI_DIR)

build: build-wasm build-cli

clean:
	go clean
	rm -f $(OUTPUT_DIR)/$(WASM_BINARY_NAME)

wasm-exec:
	cp $(WASM_EXEC_PATH) ./static

fmt:
	go fmt ./...

test: fmt
	go test ./structi/... ./svg/... -run "^Test[^B]"

test-race: fmt
	go test --race ./structi/... ./svg/... -run "^Test[^B]"

regression-test:
	go test -v --race ./structi/... -run TestRegression

benchmark:
	go test -bench=. -benchmem ./structi/... -run=^$

running_time:
	@result_path=./benchmarks/results/$$(date +%Y-%m-%d_%H-%M-%S); \
	docker build -f benchmarks/Dockerfile -t viztruct-benchmark .; \
	docker run --rm viztruct-benchmark >> $$result_path; \
	echo "Result saved to $$result_path"

serve:
	npx http-server ./static --cors

all: clean build-wasm build-cli

.PHONY: build-wasm clean fmt test test-race regression-test benchmark serve all
