#!/usr/bin/env python3
"""Simple Mock Reg.ru server for integration tests.

Supports a small subset of endpoints used by adapter: /api/regru2/zone/get_resource_records and /api/regru2/zone/update_records
"""

from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class MockHandler(BaseHTTPRequestHandler):
    def _set_headers(self):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()

    def do_POST(self):
        length = int(self.headers.get('content-length', 0))
        body = self.rfile.read(length).decode('utf-8')
        # naive routing
        if self.path.endswith('/zone/get_resource_records'):
            self._set_headers()
            resp = {"answer": {"domains": []}, "result": "success"}
            self.wfile.write(json.dumps(resp).encode('utf-8'))
            return
        if self.path.endswith('/zone/update_records'):
            self._set_headers()
            resp = {"answer": {"domains": []}, "result": "success"}
            self.wfile.write(json.dumps(resp).encode('utf-8'))
            return
        # default
        self.send_response(404)
        self.end_headers()

def run(server_class=HTTPServer, handler_class=MockHandler, port=8081):
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    print(f'Mock Reg.ru server running on port {port}')
    httpd.serve_forever()

if __name__ == '__main__':
    run()

