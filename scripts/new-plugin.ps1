param(
  [Parameter(Mandatory=$true)][string]$Key,
  [string]$Name,
  [ValidateSet('js','process-http','process-grpc','python','go-plugin')][string]$Channel = 'js'
)

$ErrorActionPreference = 'Stop'

if (-not $Name) {
  $Name = $Key
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$pluginRoot = Join-Path $repoRoot "plugins/$Key"

New-Item -ItemType Directory -Force -Path "$pluginRoot/backend/api" | Out-Null
New-Item -ItemType Directory -Force -Path "$pluginRoot/frontend/src" | Out-Null

$icon = 'Grid'
$channelDesc = @{
  'js' = 'JS 内嵌执行通道'
  'process-http' = '独立进程 HTTP 通道'
  'process-grpc' = '独立进程 gRPC 通道'
  'python' = 'Python 进程通道'
  'go-plugin' = 'Linux go-plugin 通道'
}[$Channel]

$moduleYaml = @"
name: $Name
key: $Key
version: 1.0.0
description: $channelDesc
icon: $icon
apiPrefix: /plugin/$Key
frontendEntry: /plugins/$Key/remoteEntry.js
remoteModule: ./App
permissions:
  - $Key:view
menus:
  - name: $Name
    path: /plugins/$Key
    component: RemotePluginPage
    icon: $icon
    remoteModule: ./App
"@
Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/module.yaml" -Value $moduleYaml

$routeHandler = if ($Channel -eq 'go-plugin') { 'Ping' } else { 'ping' }
$routesYaml = @"
apiPrefix: /plugin/$Key
routes:
  - method: GET
    path: /ping
    channel: $Channel
    handler: $routeHandler
"@
Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/api/routes.yaml" -Value $routesYaml

switch ($Channel) {
  'js' {
    $backendYaml = @"
channel: js
"@
    $handlersJs = @"
function ping(request, runtime) {
  return {
    channel: 'js',
    plugin: '$Key',
    message: 'pong from js handler',
    request
  }
}
"@
    Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/api/handlers.js" -Value $handlersJs
  }
  'process-http' {
    New-Item -ItemType Directory -Force -Path "$pluginRoot/backend/process-http" | Out-Null
    $backendYaml = @"
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
  routePrefix: /api/plugin/$Key
"@
    $mainGo = @"
package main

import (
  \"encoding/json\"
  \"net/http\"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc(\"/health\", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(\"ok\"))
  })
  mux.HandleFunc(\"/ping\", func(w http.ResponseWriter, r *http.Request) {
    _ = json.NewEncoder(w).Encode(map[string]any{
      \"channel\": \"process-http\",
      \"plugin\": \"$Key\",
      \"message\": \"pong from process-http\",
    })
  })
  _ = http.ListenAndServe(\":19110\", mux)
}
"@
    Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/process-http/main.go" -Value $mainGo
  }
  'process-grpc' {
    New-Item -ItemType Directory -Force -Path "$pluginRoot/backend/process-grpc" | Out-Null
    $backendYaml = @"
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
"@
    $goMod = @"
module $Key-process-grpc

go 1.21
"@
    $mainGo = @"
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
  *out = resp{StatusCode: 200, Body: map[string]any{"channel": "process-grpc", "plugin": "$Key", "handler": in.Handler, "message": "pong from process-grpc"}}
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
"@
    Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/process-grpc/go.mod" -Value $goMod
    Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/process-grpc/main.go" -Value $mainGo
  }
  'python' {
    New-Item -ItemType Directory -Force -Path "$pluginRoot/backend/process-python" | Out-Null
    $backendYaml = @"
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
  routePrefix: /api/plugin/$Key
"@
    $appPy = @"
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
            payload = {
                \"channel\": \"python\",
                \"plugin\": \"$Key\",
                \"message\": \"pong from python process\"
            }
            body = json.dumps(payload).encode('utf-8')
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
"@
    Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/process-python/app.py" -Value $appPy
  }
  'go-plugin' {
    New-Item -ItemType Directory -Force -Path "$pluginRoot/backend/go-plugin" | Out-Null
    $backendYaml = @"
channel: go-plugin
goPlugin:
  soPath: ./dist/plugin.so
"@
    $pluginGo = @"
package main

func Ping(request map[string]any, runtime map[string]any) (any, error) {
  return map[string]any{
    \"channel\": \"go-plugin\",
    \"plugin\": \"$Key\",
    \"message\": \"pong from linux go plugin\",
  }, nil
}
"@
    Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/go-plugin/plugin.go" -Value $pluginGo
    $buildSh = @"
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=\"`$(cd \"`$(dirname \"$0\")\" && pwd)\"
BACKEND_DIR=\"`$(cd \"$SCRIPT_DIR/..\" && pwd)\"
mkdir -p \"$BACKEND_DIR/dist\"
go build -buildmode=plugin -o \"$BACKEND_DIR/dist/plugin.so\" \"$SCRIPT_DIR/plugin.go\"
"@
    Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/go-plugin/build-linux.sh" -Value $buildSh
  }
}

Set-Content -Encoding UTF8 -Path "$pluginRoot/backend/api/backend.yaml" -Value $backendYaml

$remoteEntry = @"
const runtime = window.__SKOLL_VUE__ || {}
const h = runtime.h

export default {
  name: '$Key-PluginPage',
  setup() {
    return () => h('div', { style: 'padding:20px' }, [
      h('h2', '$Name'),
      h('p', 'plugin key: $Key'),
      h('p', 'backend channel: $Channel')
    ])
  }
}
"@
Set-Content -Encoding UTF8 -Path "$pluginRoot/frontend/src/remoteEntry.js" -Value $remoteEntry

Write-Host "Plugin scaffold generated: $pluginRoot"
