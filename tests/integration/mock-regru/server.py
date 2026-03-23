#!/usr/bin/env python3
"""Minimal mock Reg.ru API v2 server for integration tests.
Supports endpoints:
- /api/regru2/zone/get_resource_records
- /api/regru2/zone/update_records
- /api/regru2/zone/remove_record

Behavior:
- Expects form POST with fields: input_format, input_data (JSON string) or username/password
- If input_data contains {"simulate":"429"} -> returns 429 with Retry-After header
- If input_data contains {"simulate":"500"} -> returns 500
- Otherwise returns a simple success JSON
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import parse_qs
import json
import sys

HOST = '127.0.0.1'
PORT = 8000

class MockHandler(BaseHTTPRequestHandler):
    def _set_headers(self, code=200, headers=None):
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        if headers:
            for k, v in headers.items():
                self.send_header(k, v)
        self.end_headers()

    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length).decode('utf-8')
        params = parse_qs(body)
        # input_data may be present
        input_data = None
        if 'input_data' in params:
            try:
                input_data = json.loads(params['input_data'][0])
            except Exception:
                input_data = None
        # Simple routing
        if self.path.endswith('/zone/get_resource_records'):
            # simulate behavior
            if isinstance(input_data, dict) and input_data.get('simulate') == '429':
                self._set_headers(429, {'Retry-After':'3'})
                resp = {'result':'error','error':'rate_limited'}
                self.wfile.write(json.dumps(resp).encode())
                return
            if isinstance(input_data, dict) and input_data.get('simulate') == '500':
                self._set_headers(500)
                resp = {'result':'error','error':'server_error'}
                self.wfile.write(json.dumps(resp).encode())
                return
            # normal success
            domains = []
            if isinstance(input_data, dict) and 'domains' in input_data:
                for d in input_data['domains']:
                    domains.append({'dname': d.get('dname'), 'result':'success','rrs':[],'service_id':12345})
            self._set_headers(200)
            self.wfile.write(json.dumps({'answer':{'domains':domains}, 'result':'success'}).encode())
            return

        if self.path.endswith('/zone/update_records'):
            if isinstance(input_data, dict) and input_data.get('simulate') == '429':
                self._set_headers(429, {'Retry-After':'5'})
                self.wfile.write(json.dumps({'result':'error','error':'rate_limited'}).encode())
                return
            if isinstance(input_data, dict) and input_data.get('simulate') == '500':
                self._set_headers(500)
                self.wfile.write(json.dumps({'result':'error','error':'server_error'}).encode())
                return
            # accept action_list and echo
            domains = input_data.get('domains', []) if isinstance(input_data, dict) else []
            resp_domains = []
            for d in domains:
                resp_domains.append({'dname': d.get('dname'), 'service_id':12345, 'action_list':[{'action':'ok'}], 'result':'success'})
            self._set_headers(200)
            self.wfile.write(json.dumps({'answer':{'domains':resp_domains}, 'result':'success'}).encode())
            return

        if self.path.endswith('/zone/remove_record'):
            # simple success
            self._set_headers(200)
            self.wfile.write(json.dumps({'answer':{'result':'success'}, 'result':'success'}).encode())
            return

        # default: 404
        self._set_headers(404)
        self.wfile.write(json.dumps({'result':'error','error':'not_found'}).encode())

    def log_message(self, format, *args):
        # keep concise logs
        sys.stdout.write("%s - - [%s] %s\n" % (self.client_address[0], self.log_date_time_string(), format%args))

if __name__ == '__main__':
    server = HTTPServer((HOST, PORT), MockHandler)
    print(f"Mock Reg.ru server listening on http://{HOST}:{PORT}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print('Shutting down')
        server.server_close()

