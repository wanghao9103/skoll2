# process-grpc 通道插件开发说明

## 1. 适用场景
- 独立进程隔离
- 希望使用结构化 RPC 交互
- 对 HTTP 转发链路不满意时

## 2. 当前实现说明
当前基座中的 process-grpc 通道使用 TCP + JSON-RPC 传输，服务名/方法名约定为：
- Service: PluginGateway
- Method: Handle

请求/响应结构请参考：
- backend/internal/service/plugin_api_grpc_types.go

## 3. 关键配置

### backend/api/backend.yaml

```yaml
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
```

### backend/api/routes.yaml

```yaml
apiPrefix: /plugin/sample-grpc
routes:
  - method: GET
    path: /ping
    channel: process-grpc
    handler: Ping
```

## 4. 服务接口约定
- 入参包含 handler/method/path/query/params/body
- 返回 statusCode/body/passthrough/error

## 5. 常见问题
- address 不可达：会触发启动超时。
- handler 命名不一致：会返回 unknown handler。
