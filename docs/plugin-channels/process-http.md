# process-http 通道插件开发说明

## 1. 适用场景
- 插件已有 HTTP 服务
- 需要进程隔离
- 需要使用任意语言实现后端（只要能提供 HTTP）

## 2. 必备目录

```text
plugins/<plugin-key>/
  backend/
    module.yaml
    api/
      backend.yaml
      routes.yaml
    process-http/
      main.go  # 或其他语言实现
```

## 3. 关键配置

### backend/api/backend.yaml

```yaml
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
  routePrefix: /api/plugin/sample-http
```

### backend/api/routes.yaml

```yaml
apiPrefix: /plugin/sample-http
routes:
  - method: GET
    path: /ping
    channel: process-http
    handler: ping
```

## 4. 服务约定
- 健康检查：必须提供 healthPath，并返回 < 400。
- 路由转发：宿主会转发到独立进程的同 path。
- 响应结构：建议返回 JSON。

## 5. 常见问题
- 端口冲突：不同插件需使用不同端口。
- routePrefix 配置错误：会导致路径截断不符合预期。
