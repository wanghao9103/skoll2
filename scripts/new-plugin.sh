#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <plugin-key> [plugin-name] [channel(js|process-http|process-grpc|python|go-plugin)]"
  exit 1
fi

KEY="$1"
NAME="${2:-$KEY}"
CHANNEL="${3:-js}"

case "$CHANNEL" in
  js|process-http|process-grpc|python|go-plugin) ;;
  *)
    echo "Unsupported channel: $CHANNEL"
    exit 1
    ;;
esac

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PLUGIN_ROOT="$REPO_ROOT/plugins/$KEY"

mkdir -p "$PLUGIN_ROOT/backend/api" "$PLUGIN_ROOT/frontend/src"

cat > "$PLUGIN_ROOT/backend/module.yaml" <<EOF
name: $NAME
key: $KEY
version: 1.0.0
description: plugin scaffold
icon: Grid
apiPrefix: /plugin/$KEY
frontendEntry: /plugins/$KEY/remoteEntry.js
remoteModule: ./App
permissions:
  - $KEY:view
menus:
  - name: $NAME
    path: /plugins/$KEY
    component: RemotePluginPage
    icon: Grid
    remoteModule: ./App
EOF

HANDLER="ping"
if [ "$CHANNEL" = "go-plugin" ]; then
  HANDLER="Ping"
fi

cat > "$PLUGIN_ROOT/backend/api/routes.yaml" <<EOF
apiPrefix: /plugin/$KEY
routes:
  - method: GET
    path: /ping
    channel: $CHANNEL
    handler: $HANDLER
EOF

case "$CHANNEL" in
  js)
    cat > "$PLUGIN_ROOT/backend/api/backend.yaml" <<EOF
channel: js
EOF
    cat > "$PLUGIN_ROOT/backend/api/handlers.js" <<EOF
function ping(request, runtime) {
  return {
    channel: 'js',
    plugin: '$KEY',
    message: 'pong from js handler',
    request
  }
}
EOF
    ;;
  process-http)
    mkdir -p "$PLUGIN_ROOT/backend/process-http"
    cat > "$PLUGIN_ROOT/backend/api/backend.yaml" <<EOF
channel: process-http
process:
  startupStrategy: lazy
  startupTimeoutMs: 2000
  requestTimeoutMs: 3000
  idleRecycleSeconds: 180
  maxIdleConnsPerHost: 2
  command: go
  args:
    - run
    - ./process-http
  env: {}
  port: 19110
  healthPath: /health
  routePrefix: /api/plugin/$KEY
EOF
    cat > "$PLUGIN_ROOT/backend/process-http/main.go" <<'EOF'
package main

import (
  "encoding/json"
  "net/http"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("ok"))
  })
  mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
    _ = json.NewEncoder(w).Encode(map[string]any{
      "channel": "process-http",
      "message": "pong from process-http",
    })
  })
  _ = http.ListenAndServe(":19110", mux)
}
EOF
    ;;
  process-grpc)
    mkdir -p "$PLUGIN_ROOT/backend/process-grpc"
    cat > "$PLUGIN_ROOT/backend/api/backend.yaml" <<EOF
channel: process-grpc
grpc:
  startupStrategy: lazy
  startupTimeoutMs: 2000
  requestTimeoutMs: 3000
  idleRecycleSeconds: 180
  command: go
  args:
    - run
    - ./process-grpc
  env: {}
  address: 127.0.0.1:19112
EOF
    cat > "$PLUGIN_ROOT/backend/process-grpc/go.mod" <<EOF
module ${KEY}-process-grpc

go 1.21
EOF
    cat > "$PLUGIN_ROOT/backend/process-grpc/main.go" <<'EOF'
package main

import (
  "log"
  "net"
  "net/rpc"
  "net/rpc/jsonrpc"
)

type req struct {
  Handler string         `json:"handler"`
  Method  string         `json:"method"`
  Path    string         `json:"path"`
  Query   map[string]any `json:"query"`
  Params  map[string]any `json:"params"`
  Body    map[string]any `json:"body"`
}
type resp struct {
  StatusCode int    `json:"statusCode"`
  Body       any    `json:"body"`
  Error      string `json:"error"`
}

type impl struct{}

func (impl) Handle(in *req, out *resp) error {
  *out = resp{StatusCode: 200, Body: map[string]any{"channel": "process-grpc", "handler": in.Handler, "message": "pong from process-grpc"}}
  return nil
}

func main() {
  lis, err := net.Listen("tcp", "127.0.0.1:19112")
  if err != nil { log.Fatal(err) }
  if err := rpc.RegisterName("PluginGateway", impl{}); err != nil { log.Fatal(err) }
  for {
    conn, err := lis.Accept()
    if err != nil { continue }
    go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
  }
}
EOF
    ;;
  python)
    mkdir -p "$PLUGIN_ROOT/backend/process-python"
    cat > "$PLUGIN_ROOT/backend/api/backend.yaml" <<EOF
channel: python
process:
  startupStrategy: lazy
  startupTimeoutMs: 2000
  requestTimeoutMs: 3000
  idleRecycleSeconds: 180
  maxIdleConnsPerHost: 2
  command: python
  args:
    - ./process-python/app.py
  env: {}
  port: 19111
  healthPath: /health
  routePrefix: /api/plugin/$KEY
EOF
    cat > "$PLUGIN_ROOT/backend/process-python/app.py" <<'EOF'
from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith('/health'):
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'ok')
            return
        if self.path.startswith('/ping'):
            body = json.dumps({"channel": "python", "message": "pong from python"}).encode('utf-8')
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        self.send_response(404)
        self.end_headers()

if __name__ == '__main__':
    HTTPServer(('127.0.0.1', 19111), Handler).serve_forever()
EOF
    ;;
  go-plugin)
    mkdir -p "$PLUGIN_ROOT/backend/go-plugin"
    cat > "$PLUGIN_ROOT/backend/api/backend.yaml" <<EOF
channel: go-plugin
goPlugin:
  soPath: ./dist/plugin.so
EOF
    cat > "$PLUGIN_ROOT/backend/go-plugin/plugin.go" <<'EOF'
package main

func Ping(request map[string]any, runtime map[string]any) (any, error) {
  return map[string]any{
    "channel": "go-plugin",
    "message": "pong from linux go plugin",
  }, nil
}
EOF
    cat > "$PLUGIN_ROOT/backend/go-plugin/build-linux.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
mkdir -p "$BACKEND_DIR/dist"
go build -buildmode=plugin -o "$BACKEND_DIR/dist/plugin.so" "$SCRIPT_DIR/plugin.go"
EOF
    chmod +x "$PLUGIN_ROOT/backend/go-plugin/build-linux.sh"
    ;;
esac

cat > "$PLUGIN_ROOT/frontend/src/remoteEntry.js" <<EOF
const runtime = window.__SKOLL_VUE__ || {}
const h = runtime.h

export default {
  name: '${KEY}-PluginPage',
  setup() {
    return () => h('div', { style: 'padding:20px' }, [
      h('h2', '$NAME'),
      h('p', 'plugin key: $KEY'),
      h('p', 'backend channel: $CHANNEL')
    ])
  }
}
EOF

echo "Plugin scaffold generated: $PLUGIN_ROOT"
