# StructViz - go struct memory layout visualizer

GO = go
GOFLAGS = 
WASM_DIR = ./web
DIST_DIR = ./dist
BINARY_NAME = viztruct
SERVER_PORT = 8080

.PHONY: all check-tools test

all: check-tools clean build wasm

check-tools:
	@echo "Checking required tools..."
	@which $(GO) > /dev/null || (echo "Go is not installed. Please install Go first."; exit 1)

test:
	@echo "Running tests..."
	$(GO) test ./...

build-cli:
	$(GO) build -o $(BINARY_NAME) cmd/main.go

