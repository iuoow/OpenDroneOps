# Codex 分阶段任务

> 一次只执行一个任务。

Task 0 已完成并记录于 `TASK0_VALIDATION.md`。Task 1 已完成并记录于 `TASK1_VALIDATION.md`。
后续实现从 Task 2 开始，仍不得跳过任务顺序。

## Task 0 — 技术基线

- 核验 Go、Gin、Paho v5、coder/websocket、pgx、sqlc、Redis client、OpenTelemetry、PostgreSQL、Redis、Mosquitto 当前稳定版本。
- 建立 `DEPENDENCIES.md`：版本、许可证、来源、理由、风险。
- 决定迁移工具。
- 检查 ADR、OpenAPI、AsyncAPI、JSON Schema 和 SQL。
- 生成目录和 Makefile/Taskfile 命令设计。
- 不写业务实现。

验收：无未固定生产依赖；许可证清楚；人工确认后进入 Task 1。

## Task 1 — 工程骨架

- 初始化 Go module。
- 建立 `cmd/server`、`cmd/worker`、`cmd/simulator`、`cmd/migrate`。
- 配置加载、结构化日志、健康检查。
- PostgreSQL、Redis、Mosquitto Compose。
- CI：格式、静态检查、测试、race、构建。
- 缺少关键配置时快速失败。

## Task 2 — 领域模型与迁移

- 实现 Workspace、Device、State、Event、Alarm、Command、Outbox。
- 将 `db/schema.sql` 转换为版本化迁移。
- 唯一约束、状态版本、命令转换测试。
- Redis 暂不作为必需路径。

## Task 3 — DJI Topic 与 Envelope

- Topic Parser；
- OSD、State、Services/Reply、Events/Reply、Requests/Reply、Status；
- `tid/bid/timestamp/gateway/method/need_reply/seq`；
- 未知字段兼容；
- 非法消息隔离；
- 表驱动测试。

## Task 4 — 协议模拟器

- Gateway + Aircraft；
- OSD、State、Events、Services Reply；
- 重复、乱序、延迟、断线、重连、洪峰、非法 JSON；
- 命令成功、失败、超时；
- 1～1000 设备；
- 只通过 MQTT 通信。

## Task 5 — MQTT Ingestion Worker

- Paho v5/autopaho；
- 自动重连和订阅恢复；
- 有界 worker pool；
- 按设备哈希分片；
- 去重、隔离、指标；
- 优雅退出和 race test。

## Task 6 — 设备数字孪生

- OSD/State 标准化；
- PostgreSQL 最新状态；
- Redis 派生缓存；
- 状态版本条件更新；
- 历史事件、轨迹；
- 快照 API。

## Task 7 — WebSocket Hub

- 鉴权接口；
- 连接、订阅、取消；
- 单 writer；
- 有界队列；
- telemetry 合并；
- 心跳、超时、关闭握手；
- 慢客户端隔离；
- 快照 + cursor 恢复。

## Task 8 — 告警

- 低电量、长时间未上报、频繁断线；
- 去重、聚合、确认、恢复；
- 持久化和可重放；
- WebSocket 只作为通知。

## Task 9 — 指令与 Outbox

- `command_id` 与 `tid/bid`；
- API Idempotency-Key；
- command + outbox 同事务；
- MQTT Services Publisher；
- Reply/Event 幂等；
- 超时、审计和 orphan reply；
- 仅模拟 LOW 风险命令。

## Task 10 — Vue 控制台

- 设备、地图、状态、告警、命令；
- REST 快照 + WS 增量；
- OpenAPI 生成客户端；
- 未知枚举安全展示。

## Task 11 — 轨迹回放

- 时间范围和游标分页；
- 地图回放、倍速、暂停；
- 点数与查询范围限制；
- 性能测试。

## Task 12 — 稳定性与发布

- 1000 设备 × 1 msg/s × 1 小时；
- Broker、DB、Redis、网络故障；
- pprof、指标、Trace；
- 优雅退出；
- 安全基线；
- 可复现压测报告。

## MVP 之后

Pilot 2 WebView、真实 Dock、WPML、媒体、直播、PostGIS、多实例 WS、NATS/Kafka、ClickHouse、DRC。


## Frontend UI/UX tasks

前端的详细拆分、页面规格和质量门禁见 `UIUX_TASKS.md`、`docs/ui/` 和 `design/`。Codex 在执行原 Task 10 前必须先完成 UI Task 0～4。
