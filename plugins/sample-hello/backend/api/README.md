# Sample Hello 插件后端 API 约定

建议路由前缀：`/api/plugin/sample-hello`

## 建议接口

- `GET /api/plugin/sample-hello/records`
- `POST /api/plugin/sample-hello/records`
- `PUT /api/plugin/sample-hello/records/:id`
- `DELETE /api/plugin/sample-hello/records/:id`

## 说明

1. 安装后需启用插件，未启用状态下接口会拒绝访问。
2. 安装插件时执行 migrations/001_init.sql。
3. 升级插件时执行增量迁移脚本（002, 003...）。
4. 所有接口复用主系统 JWT，与主系统登录态一致。
