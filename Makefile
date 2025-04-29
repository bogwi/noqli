.PHONY: build test clean

# Default build target
build:
	go build -o bin/noqli ./cmd/noqli

# Run tests
test:
	go test  ./...

test-verbose:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f bin/noqli

# Install the application globally
install:
	go install ./cmd/noqli

# All targets (build and test)
all: build test 