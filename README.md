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

## 3.1 后端配置文件

后端新增配置文件：
- [backend/configs/config.yaml](backend/configs/config.yaml)

默认加载顺序：
1. 内置默认值
2. `backend/configs/config.yaml`
3. 环境变量（最高优先级）

可通过环境变量指定配置文件路径：

```powershell
$env:CONFIG_FILE = "D:/working/skoll2/backend/configs/config.yaml"
```

在配置文件中启用 MySQL 的示例：

```yaml
database:
  driver: mysql
  dsn: root:123456@tcp(127.0.0.1:3306)/skoll2?charset=utf8mb4&parseTime=True&loc=Local
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
- 后端启动时会执行数据库迁移（`schema_migration_records`），包含插件类型字段和插件配置索引修复。

## 10. VS Code 调试

仓库已内置 VS Code 调试配置：
- [.vscode/launch.json](.vscode/launch.json)
- [.vscode/tasks.json](.vscode/tasks.json)

可直接使用：
1. `Backend: Go Server` 调试后端
2. `Frontend: Chrome` 启动前端并自动打开浏览器调试
3. `Full Stack: Backend + Frontend` 一键联调

说明：VS Code 调试默认使用 `18080` 端口运行后端，避免与本地 Docker 或其他占用 `8080` 的服务冲突。

## 11. Docker 构建与运行

已提供：
- [backend/Dockerfile](backend/Dockerfile)
- [frontend/Dockerfile](frontend/Dockerfile)
- [frontend/nginx.conf](frontend/nginx.conf)
- [docker-compose.yml](docker-compose.yml)

常用命令：

```powershell
docker compose -f docker-compose.yml build --pull
docker compose -f docker-compose.yml up -d --build
docker compose -f docker-compose.yml ps
docker compose -f docker-compose.yml logs -f --tail 200
docker compose -f docker-compose.yml down
```

端口映射：
- 前端：http://localhost:5173
- 后端：http://localhost:8080

## 12. 部署脚本

已提供跨平台脚本：
- PowerShell: [scripts/deploy.ps1](scripts/deploy.ps1)
- Bash: [scripts/deploy.sh](scripts/deploy.sh)

PowerShell 示例：

```powershell
./scripts/deploy.ps1 up
./scripts/deploy.ps1 logs
./scripts/deploy.ps1 down
```

Bash 示例：

```bash
./scripts/deploy.sh up
./scripts/deploy.sh logs
./scripts/deploy.sh down
```

## 13. 插件开发方式（后端+前端+数据库）

已补充完整开发规范文档：
- [docs/插件开发规范.md](docs/插件开发规范.md)

该文档覆盖：
- 插件后端契约（生命周期、模块元数据）
- 插件前端契约（远程入口、动态路由元数据）
- 数据库迁移规范（安装/升级 SQL 管理）

## 14. 下一步（已落地示例插件）

仓库已提供一个可演示开发和使用流程的示例插件：`sample-hello`

关键文件：
- [plugins/sample-hello/backend/module.yaml](plugins/sample-hello/backend/module.yaml)
- [plugins/sample-hello/backend/migrations/001_init.sql](plugins/sample-hello/backend/migrations/001_init.sql)
- [plugins/sample-hello/backend/api/README.md](plugins/sample-hello/backend/api/README.md)
- [frontend/public/plugins/sample-hello/remoteEntry.js](frontend/public/plugins/sample-hello/remoteEntry.js)
- [plugins/sample-hello/frontend/README.md](plugins/sample-hello/frontend/README.md)

### 演示安装与使用

1. 在插件管理页安装插件：
  - `packageUrl = plugin://sample-hello`
2. 启用插件
3. 左侧会出现“示例插件”菜单
4. 点击菜单进入远程插件页面
5. 在插件管理中给 `sample-hello` 添加配置，页面会实时读取并展示

说明：插件安装会在运行时即时读取 `plugins/<key>/backend/module.yaml` 并注册菜单与前端入口，不需要重启后端服务。

### 示例插件真实接口（已实现）

- `GET /api/plugin/sample-hello/records`
- `POST /api/plugin/sample-hello/records`
- `PUT /api/plugin/sample-hello/records/:id`
- `DELETE /api/plugin/sample-hello/records/:id`

调用顺序建议：
1. 安装插件
2. 启用插件
3. 调用 records 接口
