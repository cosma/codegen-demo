#!/bin/sh

# Create necessary directories
mkdir -p generated/api generated/db

# Download dependencies
go mod download

# Generate code
make generate
