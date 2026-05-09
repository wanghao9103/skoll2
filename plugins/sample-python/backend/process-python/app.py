from http.server import BaseHTTPRequestHandler, HTTPServer
import json


class Handler(BaseHTTPRequestHandler):
    def _write_json(self, code, payload):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path.startswith('/health'):
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'ok')
            return

        if self.path.startswith('/ping'):
            self._write_json(200, {
                "channel": "python",
                "message": "pong from python process"
            })
            return

        self._write_json(404, {"message": "not found"})


if __name__ == '__main__':
    server = HTTPServer(('127.0.0.1', 19102), Handler)
    print('sample-python process listening on :19102')
    server.serve_forever()
