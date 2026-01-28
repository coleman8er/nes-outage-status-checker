# Binary name
BINARY=nes-outage-status-checker

# Build flags
LDFLAGS=-s -w

.PHONY: build clean run deps fmt vet

## build: Build the binary
build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

## clean: Remove built binary
clean:
	rm -f $(BINARY)

## deps: Download dependencies
deps:
	go mod download

## fmt: Format code
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## help: Show this help
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
