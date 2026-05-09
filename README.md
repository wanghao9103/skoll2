# Skoll2 插件化后台管理系统（基础框架）

本仓库已按需求文档完成“可运行”的基础框架，包含：
- Go 后端基座（认证、插件管理、配置中心、动态菜单接口）
- Vue3 前端管理端（登录、布局、动态菜单、插件管理页）
- 插件持久化存储（支持 MySQL，默认 SQLite 即开即跑）
- 远程插件组件动态加载入口（RemotePluginPage）

## 1. 目录结构

```text
skoll2/
  backend/        Go API 服务
  frontend/       Vue3 管理平台
  插件化后台管理系统-可开发需求文档.md
```

## 2. 环境要求
- Go >= 1.21
- Node.js = 14.16+（当前已适配）
- npm 6+

## 3. 数据库配置

默认配置：
- DB_DRIVER=sqlite
- DB_DSN=skoll2.db

即不配置数据库环境变量也可以直接运行。

### 使用 MySQL

示例（PowerShell）：

```powershell
$env:DB_DRIVER = "mysql"
$env:DB_DSN = "root:123456@tcp(127.0.0.1:3306)/skoll2?charset=utf8mb4&parseTime=True&loc=Local"
```

## 4. 后端启动

```powershell
cd backend
go mod tidy
go run ./cmd/server
```

默认监听：`http://localhost:8080`

### 已实现接口
- `GET /health`
- `POST /api/auth/login`
- `GET /api/plugin/list`
- `GET /api/plugin/config`
- `POST /api/plugin/install`
- `POST /api/plugin/config/save`
- `POST /api/plugin/upgrade`
- `POST /api/plugin/enable`
- `POST /api/plugin/disable`
- `POST /api/plugin/uninstall`
- `GET /api/menus`

## 5. 前端启动

```powershell
cd frontend
npm install --registry=https://registry.npmmirror.com
npm run dev
```

默认访问：`http://localhost:5173`

说明：
- 前端 API 基地址默认 `http://localhost:8080`
- 如需修改，可设置环境变量 `VITE_API_BASE`

## 6. 默认账号
- 用户名：admin
- 密码：admin123

## 7. 快速验证（手工）
1. 启动后端
2. 启动前端
3. 登录后台
4. 在“插件管理”输入插件包 URL（如 `https://cdn.example.com/livekit.zip`）并安装
5. 点击“配置”可保存插件配置项
6. 点击“启用”后，可在左侧菜单看到新增插件路由项
7. 新增插件菜单会进入 `RemotePluginPage`，按后端下发的 `frontendEntry` 动态加载远程组件

## 8. 插件配置中心接口示例

保存配置：

```json
POST /api/plugin/config/save
{
  "pluginKey": "livekit",
  "configs": [
    { "key": "livekit.url", "value": "wss://demo.livekit.io", "isSecret": false },
    { "key": "livekit.apiKey", "value": "LK_TEST", "isSecret": true }
  ]
}
```

读取配置：

```text
GET /api/plugin/config?pluginKey=livekit
```

## 9. 当前实现说明
- 插件信息与配置已持久化存储（SQLite/MySQL）。
- 插件菜单会附带远程组件元数据（pluginKey、frontendEntry、remoteModule）。
- 远程组件加载失败时前端会自动降级为占位展示，不影响主框架。
