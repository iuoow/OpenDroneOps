# OpenDroneOps — Codex 项目启动包

> Blueprint 0.1 · 校准日期：2026-07-26

项目标识：`OpenDroneOps`  
远端仓库：[`iuoow/OpenDroneOps`](https://github.com/iuoow/OpenDroneOps)

## 项目目标

指导 Codex 创建一个基于 Go、MQTT、WebSocket 的 DJI Cloud API 开源无人机实时运营平台。第一阶段聚焦：

- DJI MQTT 设备接入与协议适配；
- 设备数字孪生；
- WebSocket 实时推送；
- 告警中心；
- 可靠指令状态机；
- 历史轨迹与事件回放；
- 无真实硬件也能运行的协议模拟器。

本项目不直接复制 DJI 官方 Demo。官方 Demo 已终止维护，并明确不是生产级方案；外部协议只以 DJI 官方 Cloud API 文档为事实源。

## 确认后的技术栈

| 层次 | 技术 |
|---|---|
| 后端 | Go、Gin |
| MQTT | Eclipse Paho MQTT v5 Go + autopaho |
| WebSocket | coder/websocket |
| 数据 | PostgreSQL、pgx、sqlc、Redis |
| 契约 | OpenAPI、AsyncAPI、JSON Schema |
| 可观测性 | OpenTelemetry、结构化日志、Prometheus-compatible metrics |
| 前端 | Vue 3、TypeScript、MapLibre GL JS、ECharts |
| 默认 Broker | Eclipse Mosquitto |
| 可选 Broker | EMQX、HiveMQ CE |

## 架构原则

1. 模块化单体优先，MQTT Worker 和 Simulator 可独立运行。
2. DJI DTO 与 Topic 只存在于 Adapter 边界。
3. 所有队列有界、所有重试有限、所有后台任务可取消。
4. PostgreSQL 是业务事实源；Redis 只保存可重建派生状态。
5. MQTT QoS/Publish ACK 不等于设备业务成功。
6. REST 提供快照，WebSocket 提供增量。
7. 状态消息可以合并；告警和命令必须可恢复。
8. MVP 不实现 DRC、真实自动起飞、飞行控制或视频链路。
9. 模拟器必须像真实设备一样通过 MQTT 交互。
10. 依赖版本在 Task 0 中核验并固定，不使用 `latest` 或主分支。

## DJI 协议关键约束

- `thing/product/{device_sn}/osd`
- `thing/product/{device_sn}/state`
- `thing/product/{gateway_sn}/services`
- `thing/product/{gateway_sn}/services_reply`
- `thing/product/{gateway_sn}/events`
- `thing/product/{gateway_sn}/events_reply`
- 内部 `command_id` 必须映射 DJI `tid/bid`。
- DRC 是独立低延迟控制链路，不进入普通遥测 Worker。
- 设备支持范围以 DJI 当前 Product Supported 页面为准。

## 使用顺序

1. 阅读 `README.md`、`AGENTS.md` 和 `docs/development/task-sequence.md`。
2. 初始化 Git 仓库。
3. 让 Codex 读取根目录 `AGENTS.md`。
4. Task 0 的依赖、许可证、迁移工具和 MVP 边界见 `docs/development/dependencies.md`、`docs/development/license-decision.md` 与 `docs/decisions/0011-task-0-baseline-decisions.md`。
5. 先完成契约和质量验证，再逐个执行后续任务。

## 首条验收链路

```text
模拟 DJI Dock 发布 OSD
→ MQTT Worker
→ DJI Adapter
→ 标准领域事件
→ PostgreSQL/Redis 最新状态
→ WebSocket
→ 前端地图显示
```

## MVP 基准目标

- 1000 个模拟设备，每设备 1 msg/s，持续 1 小时；
- 无持续 goroutine 泄漏；
- 无无界队列；
- QoS 1 重复不会造成重复业务结果；
- Redis 清空后关键状态可从 PostgreSQL 恢复；
- 本地基准环境端到端 P95 目标小于 500ms（不是 DJI 官方指标）。


## UI/UX 设计包

新增：

- `docs/design/ui/`：页面、地图、告警、指令、回放、Pilot、无障碍和性能规格；
- `docs/design/tokens/`：JSON/CSS Tokens；
- `docs/design/wireframes/`：低保真线框；
- `docs/design/user-flows/`：告警、指令和重连流程；
- `docs/development/uiux-tasks.md`：前端任务；
- Codex 私有 Prompt 和清单保留在仓库外的本地工作区，不作为公开项目内容。

实施顺序：

```text
技术基线 → Design System → App Shell → Overview Mock
→ Snapshot + WebSocket → Devices → Alarms → Commands
→ Replay → Operations → Pilot Shell → Quality Gate
```

当前已完成 Task 0–8：工程骨架、领域/数据库边界、DJI 协议解析、协议模拟器、MQTT ingestion worker、设备数字孪生边界、WebSocket Hub 与告警生命周期。Task 8 验收记录见 `docs/development/validation/TASK8_VALIDATION.md`；Pilot Shell 属于 MVP 后阶段；MVP 主线止于 Operations，真实飞行控制和 DRC 永不从
模拟器阶段自动开放。
