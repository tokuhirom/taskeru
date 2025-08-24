.PHONY: build test fmt lint

# Default target
build:
	@echo "Building taskeru..."
	@go build -o taskeru main.go
	@echo "Build complete: ./taskeru"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Format code using gofmt via golangci-lint
fmt:
	@echo "Formatting code..."
	@gofmt -w .

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run