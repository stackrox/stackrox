.PHONY: build build-amd64 clean test

# Default target
build: build-amd64

# Build for amd64 architecture
build-amd64:
	GOOS=linux GOARCH=amd64 go build -o vm-agent-amd64 ./main.go

# Build for current architecture
build-local:
	go build -o vm-agent-local ./main.go

# Clean build artifacts
clean:
	rm -f vm-agent vm-agent-amd64 vm-agent-local

# Run tests
test:
	go test ./...

# Install dependencies
deps:
	go mod tidy
	go mod download
