# JS 通道插件开发说明

## 1. 适用场景
- 轻量业务逻辑
- 快速迭代验证
- 依赖宿主提供的 runtime（db/cache）

## 2. 必备目录

```text
plugins/<plugin-key>/
  backend/
    module.yaml
    api/
      backend.yaml
      routes.yaml
      handlers.js
  frontend/
    src/
      remoteEntry.js
```

## 3. 关键配置

### backend/api/backend.yaml

```yaml
channel: js
```

### backend/api/routes.yaml

```yaml
apiPrefix: /plugin/sample-js
routes:
  - method: GET
    path: /ping
    channel: js
    handler: ping
```

### backend/api/handlers.js

```javascript
function ping(request, runtime) {
  return {
    message: 'pong from js',
    request,
    now: Date.now()
  }
}
```

## 4. runtime 能力
- runtime.db: 基础 CRUD
- runtime.cache: set/get/del

建议：
- 只放“业务编排”与轻量逻辑
- 复杂 CPU 密集计算迁移到独立进程通道

## 5. 常见问题
- handler 名称不匹配：routes.yaml 的 handler 必须与 handlers.js 函数名一致。
- JSON body 解析失败：请求体必须是合法 JSON。
