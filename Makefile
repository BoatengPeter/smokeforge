.PHONY: build test lint fmt

build:
	go build -o bin/smokeforge ./cmd/smokeforge

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
