.PHONY: build test fmt lint

# Default target
build:
	@echo "Building taskeru..."
	@go build -o taskeru main.go
	@echo "Build complete: ./taskeru"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Clean build artifacts
clean:
	rm -f taskeru

# Format code using golangci-lint
fmt:
	@echo "Formatting code..."
	@golangci-lint fmt .

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run
