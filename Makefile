all: clean test build

build:
	@echo ">>> Building Application..."
	go build -o bin/native-ingester

test:
	@echo ">>> Running Unit Tests..."
	go test -race ./...

cover-test:
	@echo ">>> Running Tests with Coverage..."
	go test -race ./... -coverprofile=coverage.txt -covermode=atomic

clean:
	@echo ">>> Removing binaries..."
	@rm -rf bin/*
