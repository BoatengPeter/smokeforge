.PHONY: build test lint lint-fix fmt

build:
	go build -o bin/smokeforge ./cmd/smokeforge

test:
	go test ./...

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

fmt:
	golangci-lint fmt ./...
