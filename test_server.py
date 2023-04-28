from http.server import BaseHTTPRequestHandler
from http.server import HTTPServer
import json


class MockHTTPRequestHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        data = None
        if content_length := self.headers['Content-Length']:
            data = self.rfile.read(int(content_length))

        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()

        fn_name = self.path.strip('/').replace('/', '_')
        fn = getattr(self, fn_name)
        self.wfile.write(json.dumps(fn(data)).encode('utf8'))

    def api_user_token(self, _):
        return {
            "token": "0123456789012345678901234567890123456789"
        }


if __name__ == "__main__":
    server_address = ('', 45454)
    httpd = HTTPServer(server_address, MockHTTPRequestHandler)
    httpd.serve_forever()
