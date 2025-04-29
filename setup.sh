#!/bin/bash

echo "NoQLi - GO Setup Script"
echo "======================="
echo

# Check if operating system is macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Detected macOS system"
    
    # Check if Homebrew is installed
    if ! command -v brew &> /dev/null; then
        echo "Homebrew not found. Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    else
        echo "Homebrew is already installed."
    fi
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo "Go not found. Installing Go using Homebrew..."
        brew install go
    else
        echo "Go is already installed."
    fi
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Detected Linux system"
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo "Go not found. Please install Go manually using your distribution's package manager."
        echo "For example:"
        echo "  - Ubuntu/Debian: sudo apt-get install golang"
        echo "  - Fedora: sudo dnf install golang"
        echo "  - Arch Linux: sudo pacman -S go"
        echo "Or download from https://golang.org/dl/"
        exit 1
    else
        echo "Go is already installed."
    fi
else
    echo "Unsupported operating system: $OSTYPE"
    echo "Please install Go manually from https://golang.org/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}')
echo "Go version: $GO_VERSION"

# Initialize Go module and get dependencies
echo "Setting up Go module and dependencies..."
go mod tidy

echo
echo "Setup completed successfully!"
echo "You can now build and run the NoQLi application:"
echo "  1. Create a .env file with your database credentials (cp env.example .env)"
echo "  2. Build the application: go build -o noqli"
echo "  3. Run the application: ./noqli" 