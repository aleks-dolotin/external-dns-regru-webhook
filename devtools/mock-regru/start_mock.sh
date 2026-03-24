#!/usr/bin/env bash
# Start mock Reg.ru server in background with PID tracking.
# Usage: ./start_mock.sh [port]
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="${MOCK_PID_FILE:-${DIR}/mock_regru.pid}"
LOG_FILE="${MOCK_LOG_FILE:-/tmp/mock-regru.log}"
PORT="${1:-${MOCK_PORT:-8081}}"

# Stop previous instance if running.
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "Stopping previous mock server (pid $OLD_PID)..."
        kill "$OLD_PID" || true
        sleep 1
    fi
    rm -f "$PID_FILE"
fi

export MOCK_PORT="$PORT"

nohup python3 "$DIR/mock_server.py" > "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

# Wait for server to be ready (up to 5 seconds).
for i in $(seq 1 50); do
    if curl -sf "http://127.0.0.1:${PORT}/healthz" > /dev/null 2>&1; then
        echo "Mock Reg.ru server started on port ${PORT} (pid $(cat "$PID_FILE"))"
        exit 0
    fi
    sleep 0.1
done

echo "WARNING: Mock server may not have started correctly. Check $LOG_FILE" >&2
echo "PID: $(cat "$PID_FILE")"
exit 1
