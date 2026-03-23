# Mock Reg.ru Integration Tests

This folder contains a minimal mock server and basic integration tests for the Reg.ru adapter.

Quick start (macOS / zsh):

1. Install dependencies (preferably in a virtualenv):

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
pip install -r requirements.txt
```

2. Start mock server and run tests:

```bash
# start server in background
python tests/integration/mock-regru/server.py &
# run tests
pytest -q tests/integration/mock-regru/test_mock_regu.py
```

Notes
- The mock server listens on http://127.0.0.1:8000 and simulates 200/429/500 behaviors when `input_data` contains `{"simulate":"429"}` or `{"simulate":"500"}`.
- CI workflow `.github/workflows/integration-mock-regu.yml` runs these tests on GitHub Actions.

