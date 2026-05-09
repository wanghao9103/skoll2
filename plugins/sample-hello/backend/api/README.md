# Sample Hello 插件后端 API 实现

插件后端路由与接口实现放在插件目录：
- 路由定义：`plugins/sample-hello/backend/api/routes.yaml`
- 接口代码：`plugins/sample-hello/backend/api/handlers.js`
- 通道与进程参数：`plugins/sample-hello/backend/api/backend.yaml`

## 当前路由

- `GET /api/plugin/sample-hello/records`
- `POST /api/plugin/sample-hello/records`
- `PUT /api/plugin/sample-hello/records/:id`
- `DELETE /api/plugin/sample-hello/records/:id`

## 多通道

- `js`：使用 `handlers.js`（当前示例默认）
- `process-http`：独立进程 + HTTP 反向代理
- `python`：独立进程 + HTTP 反向代理（通常 command 为 python）
- `process-grpc`：预留通道（当前未实现）

## 运行时公共库（js 通道）

插件接口代码可以使用基座注入的运行时能力：

- `runtime.db.list/getById/create/updateById/deleteById`
- `runtime.cache.set/get/del`

## 低资源默认参数

当 `backend.yaml` 未显式设置时，基座使用低资源默认值：

- 启动策略：`lazy`（按需拉起）
- 启动超时：`2000ms`
- 请求超时：`3000ms`
- 空闲回收：`180s`
- 每主机空闲连接：`2`

## 说明

1. 安装后需启用插件，未启用状态下接口会拒绝访问。
2. 安装插件时执行 migrations/001_init.sql。
3. 升级插件时执行增量迁移脚本（002, 003...）。
4. 所有接口复用主系统 JWT，与主系统登录态一致。
