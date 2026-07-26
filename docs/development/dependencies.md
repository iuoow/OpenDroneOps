# OpenDroneOps 固定依赖基线

> Task 0 决策日期：2026-07-26  
> 远端归属：[`github.com/iuoow/OpenDroneOps`](https://github.com/iuoow/OpenDroneOps)

本文档固定开发基线。除非提交新的 ADR、更新本文件并完成回归验证，否则不得使用
`latest`、未固定的容器 Tag 或 Git 主分支。

## 运行时与基础设施

| 组件 | 固定版本 | 许可证 | 来源 | 选择理由 |
|---|---|---|---|---|
| Go | CI/发布 1.26.5；模块最低 1.25.0 | BSD-3-Clause | [go.dev/dl](https://go.dev/dl/) | CI 固定当前稳定版；开发机允许使用 Go 1.25.x，避免自动下载工具链 |
| PostgreSQL | 18.4 | PostgreSQL License | [postgres Docker Official Image](https://hub.docker.com/_/postgres) | 当前稳定主版本的最新小版本 |
| Redis | 7.2.14 | BSD-3-Clause | [redis Docker Official Image](https://hub.docker.com/_/redis) | 仅作为可重建派生状态，保留宽松许可证；不采用 7.4+/8.x source-available 版本 |
| Mosquitto | 2.1.2 | EPL-2.0 / EDL-1.0 | [Mosquitto 2.1.2 release](https://mosquitto.org/blog/2026/02/version-2-1-2-released/) | MQTT 5 默认 Broker，包含当前稳定修复 |
| Node.js | 24.x | MIT | [nodejs.org](https://nodejs.org/en/about/previous-releases) | 前端构建与测试运行时；以 CI 镜像中的具体小版本锁定 |
| npm | 11.5.1 | Artistic-2.0 | [npm CLI](https://github.com/npm/cli) | 前端阶段使用 lockfile 锁定传递依赖 |

## Go 依赖

| 模块 | 固定版本 | 许可证 | 来源 | 用途 |
|---|---|---|---|---|
| `github.com/gin-gonic/gin` | v1.12.0 | MIT | [Gin](https://github.com/gin-gonic/gin) | REST API 与 HTTP 中间件 |
| `github.com/eclipse-paho/paho.golang` | v0.23.0 | EPL-2.0 | [Paho Go](https://github.com/eclipse-paho/paho.golang) | MQTT 5 客户端与 autopaho 连接管理 |
| `github.com/coder/websocket` | v1.8.15 | ISC | [coder/websocket](https://github.com/coder/websocket) | WebSocket Hub |
| `github.com/jackc/pgx/v5` | v5.10.0 | MIT | [pgx](https://github.com/jackc/pgx) | PostgreSQL 驱动与连接池 |
| `github.com/sqlc-dev/sqlc` | v1.31.1 | MIT | [sqlc](https://github.com/sqlc-dev/sqlc) | 从显式 SQL 生成类型安全访问代码 |
| `github.com/redis/go-redis/v9` | v9.21.0 | BSD-2-Clause | [go-redis](https://github.com/redis/go-redis) | Redis 派生状态与缓存 |
| `go.opentelemetry.io/otel` | v1.44.0 | Apache-2.0 | [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go) | Trace、Metrics 与上下文关联 |
| `github.com/pressly/goose/v3` | v3.26.0 | MIT | [Goose](https://github.com/pressly/goose) | 版本化 SQL 迁移；兼容当前本地 Go 1.25.5 工具链 |
| `github.com/eclipse/paho.golang` | v0.23.0 | EPL-2.0 OR BSD-3-Clause | [Paho Go](https://github.com/eclipse/paho.golang) | MQTT v5 客户端与自动重连；仅在 MQTT transport 边界使用 |

## 前端依赖

| 包 | 固定版本 | 许可证 | 用途 |
|---|---|---|---|
| `vue` | 3.5.40 | MIT | UI 框架 |
| `vite` | 8.1.5 | MIT | 开发服务器与构建 |
| `typescript` | 7.0.2 | Apache-2.0 | 类型系统 |
| `vue-router` | 5.2.0 | MIT | Desktop/Pilot 路由 |
| `pinia` | 4.0.2 | MIT | 状态管理 |
| `maplibre-gl` | 6.0.0 | BSD-3-Clause | WebGL 地图 |
| `echarts` | 6.1.0 | Apache-2.0 | 趋势与回放图表 |
| `vitest` | 4.1.10 | MIT | 单元与组件测试 |
| `@playwright/test` | 1.62.0 | Apache-2.0 | 浏览器验收测试 |
| `@axe-core/playwright` | 4.12.1 | MPL-2.0 | 自动化无障碍检查 |
| `@vue/test-utils` | 2.4.11 | MIT | Vue 组件测试工具 |
| `web-vitals` | 6.0.0 | Apache-2.0 | 真实用户性能指标 |

## 许可证与供应链决策

- 项目许可证固定为 Apache-2.0；完整文本见 `LICENSE`。
- Redis 选择 7.2.14。Redis 官方说明 7.2.x 及之前仍为 BSD-3-Clause，7.4 起改为
  RSALv2/SSPL，8.x 增加 AGPLv3 选项；本项目不需要 Redis 8 的新数据类型。
- EMQX、HiveMQ CE 只作为兼容性验证，不进入默认 Compose，也不作为运行时依赖。
- MapLibre 样式 URL 必须通过环境变量配置，不绑定未授权的商业底图服务。
- 所有生产容器在发布前进一步记录镜像 digest；版本号只用于本地可读性。
- 依赖升级必须同时更新本文件、锁文件、SBOM/许可证清单，并运行完整质量门禁。
