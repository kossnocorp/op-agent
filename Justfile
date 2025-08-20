VERSION := `cat VERSION 2>/dev/null`

# Dev

build: build-client build-server

build-server:
  go build -o ./dist/op-agent ./cmd/op-agent

build-client:
  go build -o ./dist/op-agent-client ./cmd/op-agent-client

# All

build-all: build-linux build-darwin build-windows

# Linux

build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
  GOOS=linux GOARCH=amd64 go build -o ./dist/op-agent-{{VERSION}}-linux-amd64 ./cmd/op-agent
  GOOS=linux GOARCH=amd64 go build -o ./dist/op-agent-client-{{VERSION}}-linux-amd64 ./cmd/op-agent-client

build-linux-arm64:
  GOOS=linux GOARCH=arm64 go build -o ./dist/op-agent-{{VERSION}}-linux-arm64 ./cmd/op-agent
  GOOS=linux GOARCH=arm64 go build -o ./dist/op-agent-client-{{VERSION}}-linux-arm64 ./cmd/op-agent-client

# macOS

build-darwin: build-darwin-amd64 build-darwin-arm64

build-darwin-amd64:
  GOOS=darwin GOARCH=amd64 go build -o ./dist/op-agent-{{VERSION}}-darwin-amd64 ./cmd/op-agent
  GOOS=darwin GOARCH=amd64 go build -o ./dist/op-agent-client-{{VERSION}}-darwin-amd64 ./cmd/op-agent-client

build-darwin-arm64:
  GOOS=darwin GOARCH=arm64 go build -o ./dist/op-agent-{{VERSION}}-darwin-arm64 ./cmd/op-agent
  GOOS=darwin GOARCH=arm64 go build -o ./dist/op-agent-client-{{VERSION}}-darwin-arm64 ./cmd/op-agent-client

# Windows

build-windows: build-windows-amd64 build-windows-arm64

build-windows-amd64:
  GOOS=windows GOARCH=amd64 go build -o ./dist/op-agent-{{VERSION}}-windows-amd64.exe ./cmd/op-agent
  GOOS=windows GOARCH=amd64 go build -o ./dist/op-agent-client-{{VERSION}}-windows-amd64.exe ./cmd/op-agent-client

build-windows-arm64:
  GOOS=windows GOARCH=arm64 go build -o ./dist/op-agent-{{VERSION}}-windows-arm64.exe ./cmd/op-agent
  GOOS=windows GOARCH=arm64 go build -o ./dist/op-agent-client-{{VERSION}}-windows-arm64.exe ./cmd/op-agent-client

# Misc

clean:
  rm -rf ./dist/*

version:
  @echo {{VERSION}}