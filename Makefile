APP_NAME   := vsixctl
MODULE     := github.com/E-n-d-l-e-s-s-A-I/vsixctl
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -X $(MODULE)/cmd.Version=$(VERSION)
BUILD_DIR  := build

.PHONY: build test lint fmt vet check clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .

test:
	go test ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run

fmt:
	gofmt -l -w .

vet:
	go vet ./...

## check — запускает fmt, vet, lint и тесты
check: fmt vet lint test

clean:
	rm -rf $(BUILD_DIR)
