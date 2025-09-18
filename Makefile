.PHONY: run build lint test


run:
go run ./cmd/server


build:
go build -o bin/server ./cmd/server


lint:
@echo "(optional) add golangci-lint here"


test:
go test ./...