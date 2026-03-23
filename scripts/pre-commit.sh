#!/usr/bin/env bash
set -e

echo "=== Pre-commit: go vet ==="
go vet ./...

echo "=== Pre-commit: go test ==="
go test ./...

# golangci-lint — запускаем только если установлен
if command -v golangci-lint &>/dev/null; then
  echo "=== Pre-commit: golangci-lint ==="
  golangci-lint run ./...
else
  echo "=== Pre-commit: golangci-lint not found, skipping (install: https://golangci-lint.run/usage/install/) ==="
fi

echo "=== Pre-commit: all checks passed ✅ ==="

