#!/usr/bin/env python3
"""Mock Reg.ru API v2 server for integration tests.

Supports endpoints:
- POST /api/regru2/zone/get_resource_records
- POST /api/regru2/zone/update_records
- POST /api/regru2/zone/remove_record
- DELETE /reset  (admin: clears all in-memory records)
- GET /healthz   (admin: health check)

Stateful mode:
  In-memory record storage keyed by (zone, subdomain, rectype).
  update_records with add_* actions stores records.
  get_resource_records returns stored records.
  remove_record deletes stored records.

Error simulation:
  - MOCK_SIMULATE_429=true or ?simulate_429=true: Nth request returns 429 with Retry-After.
  - MOCK_SIMULATE_5XX=true: intermittent 500 responses.

Authentication:
  - MOCK_USERNAME / MOCK_PASSWORD: when set, validates username/password in input_data.
"""

from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import parse_qs, urlparse
import json
import os
import sys
import threading

# ---------------------------------------------------------------------------
# Configuration from environment
# ---------------------------------------------------------------------------
HOST = os.getenv("MOCK_HOST", "0.0.0.0")
PORT = int(os.getenv("MOCK_PORT", "8081"))

MOCK_USERNAME = os.getenv("MOCK_USERNAME", "")
MOCK_PASSWORD = os.getenv("MOCK_PASSWORD", "")

SIMULATE_429 = os.getenv("MOCK_SIMULATE_429", "").lower() == "true"
SIMULATE_5XX = os.getenv("MOCK_SIMULATE_5XX", "").lower() == "true"

# Every Nth request triggers 429 (when simulation is on).
SIMULATE_429_EVERY_N = int(os.getenv("MOCK_SIMULATE_429_EVERY_N", "3"))
# Every Nth request triggers 500 (when simulation is on).
SIMULATE_5XX_EVERY_N = int(os.getenv("MOCK_SIMULATE_5XX_EVERY_N", "4"))

# ---------------------------------------------------------------------------
# Shared state (thread-safe)
# ---------------------------------------------------------------------------
_lock = threading.Lock()
# records: dict[ (zone, subdomain, rectype) ] -> { "content": str, "prio": str, "state": str }
_records: dict[tuple[str, str, str], dict] = {}
# Global request counter for error simulation.
_request_counter = 0


def _next_request_id() -> int:
    global _request_counter
    with _lock:
        _request_counter += 1
        return _request_counter


def _reset_state():
    global _records, _request_counter
    with _lock:
        _records.clear()
        _request_counter = 0


# ---------------------------------------------------------------------------
# Helper: build Reg.ru-compatible response envelope
# ---------------------------------------------------------------------------
def _success_response(domains: list[dict]) -> dict:
    return {"result": "success", "answer": {"domains": domains}}


def _error_response(error_code: str, error_text: str) -> dict:
    return {"result": "error", "error_code": error_code, "error_text": error_text}


def _domain_success(dname: str, rrs: list | None = None, action_list: list | None = None) -> dict:
    d: dict = {"dname": dname, "result": "success", "service_id": 12345}
    if rrs is not None:
        d["rrs"] = rrs
    if action_list is not None:
        d["action_list"] = action_list
    return d


# ---------------------------------------------------------------------------
# Record helpers
# ---------------------------------------------------------------------------
def _get_records_for_zone(zone: str) -> list[dict]:
    """Return all resource records stored for a given zone."""
    with _lock:
        result = []
        for (z, sub, rtype), vals in _records.items():
            if z == zone:
                result.append({
                    "subname": sub,
                    "rectype": rtype,
                    "content": vals.get("content", ""),
                    "prio": vals.get("prio", ""),
                    "state": vals.get("state", "A"),
                })
        return result


def _extract_content_from_action(action_entry: dict, action_name: str) -> str:
    """Extract the content value from an action entry based on action type."""
    if action_name in ("add_alias", "add_aaaa"):
        return action_entry.get("ipaddr", "")
    elif action_name == "add_cname":
        return action_entry.get("canonical_name", "")
    elif action_name == "add_txt":
        return action_entry.get("text", "")
    return action_entry.get("content", "")


def _apply_action(zone: str, action_entry: dict) -> dict:
    """Apply a single action from action_list. Returns an action result dict."""
    action_name = action_entry.get("action", "")
    subdomain = action_entry.get("subdomain", "")

    if action_name == "remove_record":
        rectype = action_entry.get("record_type", "")
        key = (zone, subdomain, rectype)
        with _lock:
            if key in _records:
                del _records[key]
        return {"action": action_name, "result": "success"}

    # add_alias, add_aaaa, add_cname, add_txt
    rectype_map = {
        "add_alias": "A",
        "add_aaaa": "AAAA",
        "add_cname": "CNAME",
        "add_txt": "TXT",
    }
    rectype = rectype_map.get(action_name, "")
    if not rectype:
        return {"action": action_name, "result": "error"}

    content = _extract_content_from_action(action_entry, action_name)
    key = (zone, subdomain, rectype)
    with _lock:
        _records[key] = {
            "content": content,
            "prio": action_entry.get("prio", ""),
            "state": "A",
        }
    return {"action": action_name, "result": "success"}


# ---------------------------------------------------------------------------
# HTTP Handler
# ---------------------------------------------------------------------------
class MockHandler(BaseHTTPRequestHandler):
    """Handler for mock Reg.ru API v2 endpoints."""

    def _send_json(self, code: int, data: dict, extra_headers: dict | None = None):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        if extra_headers:
            for k, v in extra_headers.items():
                self.send_header(k, v)
        self.end_headers()
        self.wfile.write(json.dumps(data).encode("utf-8"))

    def _parse_input_data(self) -> dict | None:
        """Read POST body, parse form-encoded input_data JSON."""
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length).decode("utf-8")
        params = parse_qs(body)
        if "input_data" in params:
            try:
                return json.loads(params["input_data"][0])
            except (json.JSONDecodeError, IndexError):
                return None
        return None

    def _check_auth(self, input_data: dict | None) -> bool:
        """Validate authentication if MOCK_USERNAME/MOCK_PASSWORD are configured."""
        if not MOCK_USERNAME and not MOCK_PASSWORD:
            return True  # auth not configured — allow all
        if input_data is None:
            return False
        return (
            input_data.get("username") == MOCK_USERNAME
            and input_data.get("password") == MOCK_PASSWORD
        )

    def _should_simulate_429(self) -> bool:
        """Check query param or env var for 429 simulation."""
        parsed = urlparse(self.path)
        qs = parse_qs(parsed.query)
        if qs.get("simulate_429", [""])[0].lower() == "true":
            return True
        if SIMULATE_429:
            req_id = _next_request_id()
            return req_id % SIMULATE_429_EVERY_N == 0
        return False

    def _should_simulate_5xx(self) -> bool:
        """Check env var for 5xx simulation."""
        if SIMULATE_5XX:
            req_id = _next_request_id()
            return req_id % SIMULATE_5XX_EVERY_N == 0
        return False

    # ---- Admin endpoints ----

    def do_DELETE(self):
        if self.path.rstrip("/") == "/reset":
            _reset_state()
            self._send_json(200, {"status": "ok", "message": "state reset"})
            return
        self.send_response(404)
        self.end_headers()

    def do_GET(self):
        if self.path.rstrip("/") == "/healthz":
            self._send_json(200, {"status": "ok"})
            return
        self.send_response(404)
        self.end_headers()

    # ---- API endpoints ----

    def do_POST(self):
        path = urlparse(self.path).path
        input_data = self._parse_input_data()

        # --- Error simulation (before auth, like a real overloaded server) ---
        if self._should_simulate_429():
            self._send_json(429, _error_response("RATE_LIMITED", "Too many requests"), {"Retry-After": "1"})
            return

        if self._should_simulate_5xx():
            self._send_json(500, _error_response("INTERNAL_ERROR", "Internal server error"))
            return

        # --- Auth validation ---
        if not self._check_auth(input_data):
            self._send_json(200, _error_response("INVALID_AUTH", "Authentication failed"))
            return

        # --- Routing ---
        if path.endswith("/zone/get_resource_records"):
            self._handle_get_resource_records(input_data)
        elif path.endswith("/zone/update_records"):
            self._handle_update_records(input_data)
        elif path.endswith("/zone/remove_record"):
            self._handle_remove_record(input_data)
        else:
            self._send_json(404, _error_response("UNKNOWN_ENDPOINT", f"Unknown endpoint: {path}"))

    def _handle_get_resource_records(self, input_data: dict | None):
        domains_in = (input_data or {}).get("domains", [])
        domains_out = []
        for d in domains_in:
            dname = d.get("dname", "")
            rrs = _get_records_for_zone(dname)
            domains_out.append(_domain_success(dname, rrs=rrs))
        self._send_json(200, _success_response(domains_out))

    def _handle_update_records(self, input_data: dict | None):
        domains_in = (input_data or {}).get("domains", [])
        domains_out = []
        for d in domains_in:
            dname = d.get("dname", "")
            action_list = d.get("action_list", [])
            results = []
            for action_entry in action_list:
                result = _apply_action(dname, action_entry)
                results.append(result)
            domains_out.append(_domain_success(dname, action_list=results))
        self._send_json(200, _success_response(domains_out))

    def _handle_remove_record(self, input_data: dict | None):
        if input_data is None:
            self._send_json(200, _error_response("INVALID_INPUT", "Missing input_data"))
            return

        domains_in = input_data.get("domains", [])
        subdomain = input_data.get("subdomain", "")
        record_type = input_data.get("record_type", "")

        domains_out = []
        for d in domains_in:
            dname = d.get("dname", "")
            key = (dname, subdomain, record_type)
            with _lock:
                if key in _records:
                    del _records[key]
            domains_out.append(_domain_success(dname))
        self._send_json(200, _success_response(domains_out))

    def log_message(self, format, *args):
        sys.stdout.write(
            "%s - [%s] %s\n"
            % (self.client_address[0], self.log_date_time_string(), format % args)
        )


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def run(host: str = HOST, port: int = PORT):
    server = HTTPServer((host, port), MockHandler)  # type: ignore[arg-type]
    print(f"Mock Reg.ru server running on http://{host}:{port}")
    print(f"  Auth: {'enabled' if MOCK_USERNAME else 'disabled'}")
    print(f"  429 simulation: {'enabled' if SIMULATE_429 else 'disabled'}")
    print(f"  5xx simulation: {'enabled' if SIMULATE_5XX else 'disabled'}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down mock server")
        server.server_close()


if __name__ == "__main__":
    run()
