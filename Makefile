.PHONY: build test lint lint-fix fmt

build:
	go build -o bin/smokeforge ./cmd/smokeforge

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

fmt:
	golangci-lint fmt ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean
