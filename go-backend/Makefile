.PHONY: build run test lint fmt vet clean migrate

BINARY=omniroute
BUILD_DIR=./bin
CMD_DIR=./cmd/server

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

run:
	go run $(CMD_DIR)/main.go

dev:
	PORT=20128 go run $(CMD_DIR)/main.go

test:
	go test ./... -v -race

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

migrate:
	go run $(CMD_DIR)/main.go --migrate-only

tidy:
	go mod tidy

.DEFAULT_GOAL := build
