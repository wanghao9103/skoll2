# Sample Hello 插件前端说明

插件前端源码统一放在：
- `plugins/sample-hello/frontend/src/remoteEntry.js`

开发和构建时会自动同步生成：
- `frontend/public/plugins/sample-hello/remoteEntry.js`

主应用通过该静态入口在 `RemotePluginPage` 中动态 import。

## 本地演示流程

1. 启动后端（建议端口 18080）
2. 启动前端（5173）
3. 登录后，在插件管理安装：
   - packageUrl: `https://example.com/sample-hello.zip`
4. 启用插件
5. 菜单出现“示例插件”，点击即可看到远程页面

## 配置联动演示

可在插件管理中点击“配置”，为 `sample-hello` 添加配置项。
页面会通过 `/api/plugin/config?pluginKey=sample-hello` 拉取并渲染。
