# Sample Hello 插件后端 API 实现

插件后端路由与接口实现放在插件目录：
- 路由定义：`plugins/sample-hello/backend/api/routes.yaml`
- 接口代码：`plugins/sample-hello/backend/api/handlers.js`

## 当前路由

- `GET /api/plugin/sample-hello/records`
- `POST /api/plugin/sample-hello/records`
- `PUT /api/plugin/sample-hello/records/:id`
- `DELETE /api/plugin/sample-hello/records/:id`

## 运行时公共库

插件接口代码可以使用基座注入的运行时能力：

- `runtime.db.list/getById/create/updateById/deleteById`
- `runtime.cache.set/get/del`

## 说明

1. 安装后需启用插件，未启用状态下接口会拒绝访问。
2. 安装插件时执行 migrations/001_init.sql。
3. 升级插件时执行增量迁移脚本（002, 003...）。
4. 所有接口复用主系统 JWT，与主系统登录态一致。
