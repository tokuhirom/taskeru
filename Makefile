.PHONY: all build test clean run install lint coverage fmt help

# デフォルトターゲット
all: build

# ビルド
build:
	@echo "Building taskeru..."
	@go build -o taskeru main.go
	@echo "Build complete: ./taskeru"

# テスト実行
test:
	@echo "Running tests..."
	@go test -v ./...

# カバレッジ
coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@echo ""
	@echo "Detailed coverage report:"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

# フォーマット
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

# リント (golangci-lintがインストールされている場合)
lint:
	@if command -v golangci-lint > /dev/null 2>&1; then \
		echo "Running linter..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "Running go vet instead..."; \
		go vet ./...; \
	fi

# 依存関係の更新
deps:
	@echo "Updating dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

# クリーン
clean:
	@echo "Cleaning..."
	@rm -f taskeru
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# インストール
install: build
	@echo "Installing taskeru to $(GOPATH)/bin..."
	@go install
	@echo "Installation complete"

# 開発用の実行
run: build
	@./taskeru

# デバッグビルド
debug:
	@echo "Building with debug symbols..."
	@go build -gcflags="all=-N -l" -o taskeru main.go
	@echo "Debug build complete"

# リリースビルド（最適化あり）
release:
	@echo "Building release version..."
	@go build -ldflags="-s -w" -o taskeru main.go
	@echo "Release build complete"

# クロスコンパイル
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build -o taskeru-linux-amd64 main.go
	@GOOS=darwin GOARCH=amd64 go build -o taskeru-darwin-amd64 main.go
	@GOOS=darwin GOARCH=arm64 go build -o taskeru-darwin-arm64 main.go
	@GOOS=windows GOARCH=amd64 go build -o taskeru-windows-amd64.exe main.go
	@echo "Cross-compilation complete"

# ベンチマーク
bench:
	@echo "Running benchmarks..."
	@go test -bench=. ./...

# 静的解析
check: fmt lint test
	@echo "All checks passed!"

# ヘルプ
help:
	@echo "Available targets:"
	@echo "  make build        - Build the application"
	@echo "  make test         - Run tests"
	@echo "  make coverage     - Run tests with coverage report"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linter"
	@echo "  make deps         - Update dependencies"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install      - Install to GOPATH/bin"
	@echo "  make run          - Build and run"
	@echo "  make debug        - Build with debug symbols"
	@echo "  make release      - Build optimized release version"
	@echo "  make cross-compile - Build for multiple platforms"
	@echo "  make bench        - Run benchmarks"
	@echo "  make check        - Run fmt, lint, and test"
	@echo "  make help         - Show this help message"