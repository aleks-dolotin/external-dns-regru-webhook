#!/usr/bin/env bash
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
python3 "$DIR/mock_server.py" &
echo $! > "$DIR/mock_regru.pid"
echo "Mock Reg.ru server started (pid $(cat $DIR/mock_regru.pid))"

