BINARY_NAME=prompt-cache
PROVIDER?=openai

.PHONY: all build run test benchmark clean help

all: build

help:
	@echo "PromptCache - Makefile commands:"
	@echo ""
	@echo "  make build         - Build the binary"
	@echo "  make run           - Build and run the server"
	@echo "  make test          - Run unit tests"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make benchmark     - Run benchmark suite"
	@echo "  make bench-go      - Run Go benchmarks"
	@echo "  make clean         - Clean build artifacts and data"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run with Docker Compose"
	@echo ""
	@echo "Environment variables:"
	@echo "  PROVIDER          - Embedding provider (openai|mistral|claude)"
	@echo "  OPENAI_API_KEY    - OpenAI API key"
	@echo "  MISTRAL_API_KEY   - Mistral API key"
	@echo "  ANTHROPIC_API_KEY - Anthropic API key"
	@echo "  VOYAGE_API_KEY    - Voyage AI API key (for Claude)"

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) ./cmd/api

run:
	@./scripts/run.sh

test:
	@echo "Running tests..."
	@go test ./...

test-verbose:
	@echo "Running tests (verbose)..."
	@go test ./... -v

benchmark:
	@echo "Running benchmark suite..."
	@./scripts/benchmark.sh

bench-go:
	@echo "Running Go benchmarks..."
	@go test ./internal/semantic/... -bench=. -benchmem

clean:
	@echo "Cleaning up..."
	@go clean
	@rm -f $(BINARY_NAME)
	@rm -rf badger_data data
	@rm -f server.log

docker-build:
	@echo "Building Docker image..."
	@docker build -t prompt-cache:latest .

docker-run:
	@echo "Starting with Docker Compose..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping Docker Compose..."
	@docker-compose down

