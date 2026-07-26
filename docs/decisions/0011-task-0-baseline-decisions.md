# ADR 0011: Task 0 基线与范围决策

- 状态：Accepted
- 日期：2026-07-26
- 决策者：项目默认维护者（根据用户授权自动裁决）

## 背景

蓝图在进入工程实现前仍有许可证、依赖版本、迁移工具、登录、多租户、
Pilot 2、原始消息保留和远端仓库等未决问题。继续编码会导致契约和实现反复变更。

## 决策

1. 项目名称为 `OpenDroneOps`，预期远端为 `iuoow/OpenDroneOps`。远端仓库当前不存在，
   不在 Task 0 自动创建。
2. 项目许可证采用 Apache-2.0。
3. Task 0 固定 CI/发布 Go 1.26.5（模块最低 Go 1.25.0）、PostgreSQL 18.4、Redis 7.2.14、Mosquitto 2.1.2，
   以及 `docs/development/dependencies.md` 中列出的 Go/Node 依赖。
4. 数据库迁移采用 Goose，使用显式 SQL；不引入 ORM。
5. 后端保持多 Workspace 隔离；Desktop UI 一次只操作一个 Workspace。
6. MVP 不实现真实用户登录和商业 IAM。开发环境使用明确的本地 Actor，
   但所有 API/WS 仍强制携带 `workspace_id` 边界，生产鉴权留在后续安全阶段。
7. 原始正常 OSD 保留 7 天或采样；错误、未知和隔离消息保留 30 天；
   指令与审计保留 1 年；具体部署可通过配置缩短。
8. Pilot 2 Shell 仅保留规格和 Mock Bridge，列为 MVP 之后，不进入当前后端/前端主线。
9. MVP 只允许模拟器提供的 LOW 风险 Method（`sim_status_refresh`），
   不实现 DRC、起飞、降落、返航或飞行控制。
10. WebSocket、领域事件和 API 统一使用 `workspace_id`，不再使用 `tenant_id`。
11. 增加全局事件恢复 API，并补齐 DJI MQTT 契约中的 requests、replies 和 status topic。

## 后果

- Task 1 可以直接按固定基线创建 Go module、Compose 和 CI。
- Redis 版本不是最新主线，但许可证和 MVP 的缓存用途更稳定；升级 Redis 需要单独评估许可证。
- 登录、Pilot 2 和真实设备接入不会阻塞本地数字孪生闭环。
- API/Schema/AsyncAPI 先统一，再生成客户端和实现 WebSocket 恢复。
