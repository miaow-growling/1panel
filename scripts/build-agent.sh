#!/usr/bin/env bash
# Build lightweight Worker Agent binary
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

echo "=== Building 1Panel Lightweight Worker Agent ==="
mkdir -p build

CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X 'github.com/1Panel-dev/1Panel/backend/constant.Version=v1.10-agent'" -o build/1panel-agent ./cmd/server

echo "Agent built successfully: build/1panel-agent"
ls -lh build/1panel-agent
