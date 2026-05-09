# python 通道插件开发说明

## 1. 适用场景
- 希望用 Python 快速开发插件后端
- 适合工具型、数据处理型插件

## 2. 必备目录

```text
plugins/<plugin-key>/
  backend/
    module.yaml
    api/
      backend.yaml
      routes.yaml
    process-python/
      app.py
```

## 3. 关键配置

### backend/api/backend.yaml

```yaml
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
  routePrefix: /api/plugin/sample-python
```

## 4. 服务约定
- 对外提供 HTTP 接口
- 提供 /health 健康检查
- 路由与 process-http 一致

## 5. 常见问题
- Python 环境缺失：command 执行失败。
- 依赖包未安装：进程启动后立刻退出。
