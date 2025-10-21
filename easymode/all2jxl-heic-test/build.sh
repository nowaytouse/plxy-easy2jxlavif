#!/bin/bash

# Build script for all2jxl converter

echo "Building All2JXL Converter..."

# Ensure we're in the project root
cd "$(dirname "$0")"

# Initialize Go modules
echo "Initializing Go modules..."
go mod tidy

# Build the project
echo "Compiling source code..."
go build -o bin/all2jxl main.go

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Executable located at: bin/all2jxl"
    echo ""
    echo "To run the converter:"
    echo "  ./bin/all2jxl [OPTIONS] <input_directory> [output_directory]"
    echo ""
    echo "For help:"
    echo "  ./bin/all2jxl --help"
else
    echo "Build failed!"
    exit 1
fi