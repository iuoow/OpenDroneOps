# Task 0 验证记录

验证日期：2026-07-26

## 已完成

- 本地 Git 仓库已初始化，默认分支为 `main`。
- 预期远端确定为 `github.com/iuoow/OpenDroneOps`；GitHub API 返回 404，
  因此未自动创建远端仓库或配置推送。
- Go 模块版本通过 Go Module Proxy 查询。
- Node 包版本和许可证通过官方 npm registry 查询。
- JSON 文件 4/4 解析通过。
- Docker Compose 配置解析通过，使用本地占位环境变量。
- `deployment/mosquitto.conf` 已补齐为仅供本地开发的匿名配置。
- OpenAPI、AsyncAPI、WebSocket Schema 的 Workspace 命名和事件恢复契约已对齐。
- 未生成业务代码、Go module、Node 应用或真实 DJI 凭证。

## 固定版本

完整清单见 `docs/development/dependencies.md`。核心版本：

```text
Go                 CI/发布 1.26.5；本地模块最低 1.25.0
Gin                v1.12.0
Paho Go            v0.23.0
coder/websocket    v1.8.15
pgx                v5.10.0
sqlc               v1.31.1
go-redis           v9.21.0
OpenTelemetry      v1.44.0
Goose              v3.27.3
Vue                3.5.40
Vite               8.1.5
TypeScript         7.0.2
MapLibre GL JS     6.0.0
ECharts            6.1.0
PostgreSQL         18.4
Redis              7.2.14
Mosquitto          2.1.2
```

## 尚未执行

- 尚未执行 `go test`、`go test -race`、构建或集成测试，因为项目还没有 Go module。
- 尚未启动容器；Docker Desktop 当前读取用户配置时有权限警告。
- 尚未执行完整 YAML/OpenAPI linter；当前环境没有安装独立 YAML 解析器。
- 当时尚未创建 GitHub 远端仓库；当前远端已创建，但后续整理完成前仍未推送。

## 下一步（Task 0 完成时）

进入路线图中的 Task 1：只建立工程骨架、配置加载、健康检查、
Compose/CI 和快速失败路径，不实现 MQTT、领域模型或业务页面。该任务已在
`docs/development/validation/TASK1_VALIDATION.md` 中完成。
