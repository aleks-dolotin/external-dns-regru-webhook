import subprocess
import time
import requests
import os
import signal
from pathlib import Path
import pytest


def _start_mock_server():
    # server.py is located at tests/integration/mock-regru/server.py relative to repo root
    repo_root = Path(__file__).resolve().parents[2]
    server_path = repo_root / 'tests' / 'integration' / 'mock-regru' / 'server.py'
    log_path = Path('/tmp/mock-regu-gherkin.log')
    log_fd = open(log_path, 'ab')
    proc = subprocess.Popen(['python3', str(server_path)], cwd=str(repo_root), stdout=log_fd, stderr=log_fd)

    # wait for server to accept connections
    base = 'http://127.0.0.1:8000'
    for _ in range(20):
        try:
            r = requests.get(base, timeout=0.5)
            break
        except Exception:
            time.sleep(0.2)
    else:
        # failed to start
        proc.terminate()
        log_fd.close()
        raise RuntimeError('Mock Reg.ru server failed to start')

    return proc, log_fd


@pytest.fixture(scope='session', autouse=True)
def mock_regu_server():
    """Start and stop the mock Reg.ru server for Gherkin-derived tests.

    Note: This starts the existing mock server implementation in
    `tests/integration/mock-regru/server.py`. Tests are scaffolded and
    currently skipped because the adapter under test is not implemented yet.
    """
    proc, log_fd = _start_mock_server()
    try:
        yield 'http://127.0.0.1:8000/api/regru2'
    finally:
        try:
            proc.terminate()
            proc.wait(timeout=2)
        except Exception:
            proc.kill()
        log_fd.close()

