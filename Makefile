.PHONY: build test clean

# Default build target
build:
	go build -o bin/noqli ./cmd/noqli

# Run tests with timing information and no caching
test:
	go test -count=1 -race -timeout=30s ./...

# Run tests with verbose output, timing information, and no caching
test-verbose:
	go test -v -count=1 -race -timeout=30s ./...

# Run tests with coverage report
test-coverage:
	go test -count=1 -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run benchmarks
benchmark:
	go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	rm -f bin/noqli
	rm -f coverage.out

# Install the application globally
install:
	go install ./cmd/noqli

# All targets (build and test)
all: build test 