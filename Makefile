.PHONY: build run lint test clean

# Build the application
build:
	go build -o tracker main.go

# Run the application
run:
	go run main.go

# Run the linter
lint:
	golangci-lint run --enable=errcheck,govet,ineffassign,staticcheck,unused,misspell,gocritic,gosec

# Run tests (if any)
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Clean build artifacts
clean:
	rm -f tracker

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run linter and fix auto-fixable issues
lint-fix:
	golangci-lint run --fix

# Check if code compiles
check:
	go build -o /dev/null main.go