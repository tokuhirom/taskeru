.PHONY: all test clean

# Default target - build the binary
all: taskeru

# Build the binary
taskeru:
	go build -o taskeru

# Run all tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f taskeru

fmt:
	golangci-lint fmt .
