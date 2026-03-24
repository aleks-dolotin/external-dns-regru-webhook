#!/usr/bin/env bash
# Stop mock Reg.ru server using stored PID file.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="${MOCK_PID_FILE:-${DIR}/mock_regru.pid}"

if [ ! -f "$PID_FILE" ]; then
    echo "No PID file found at $PID_FILE — server may not be running."
    exit 0
fi

PID=$(cat "$PID_FILE")

if kill -0 "$PID" 2>/dev/null; then
    echo "Stopping mock Reg.ru server (pid $PID)..."
    kill "$PID"
    sleep 1
    # Force kill if still running.
    if kill -0 "$PID" 2>/dev/null; then
        kill -9 "$PID" 2>/dev/null || true
    fi
    echo "Mock server stopped."
else
    echo "Process $PID is not running."
fi

rm -f "$PID_FILE"
