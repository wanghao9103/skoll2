# go-plugin 通道插件开发说明

## 1. 适用场景
- 追求 Go 原生调用性能
- 可接受 Linux-only 约束

## 2. 平台限制
- go-plugin 通道仅支持 Linux。
- Windows/macOS 下会命中 stub 并返回不支持错误。

## 3. 必备目录

```text
plugins/<plugin-key>/
  backend/
    module.yaml
    api/
      backend.yaml
      routes.yaml
    go-plugin/
      plugin.go
      build-linux.sh
    dist/
      plugin.so
```

## 4. 关键配置

### backend/api/backend.yaml

```yaml
channel: go-plugin
goPlugin:
  soPath: ./dist/plugin.so
```

### plugin 导出函数
导出函数签名：

```go
func Ping(request map[string]any, runtime map[string]any) (any, error)
```

routes.yaml 的 handler 需与导出函数名一致（例如 Ping）。

## 5. 构建示例

```bash
go build -buildmode=plugin -o ./dist/plugin.so ./go-plugin/plugin.go
```

## 6. 常见问题
- ABI 不匹配：宿主与插件 Go 版本不一致。
- soPath 错误：运行时无法加载 .so。
