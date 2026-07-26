# WebSocket 契约

## 用途

推送在线、遥测、状态、告警、命令和系统通知。WebSocket 不是唯一事实源，最终状态可由 REST 查询。

## Envelope

```json
{
  "event_id": "01...",
  "type": "device.telemetry",
  "schema_version": "1.0",
  "workspace_id": "workspace-001",
  "aggregate_id": "device-001",
  "occurred_at": "2026-07-26T00:00:00Z",
  "sequence": 100,
  "data": {}
}
```

## 事件

`session.ready`、`session.error`、`device.online`、`device.offline`、`device.telemetry`、`device.state_changed`、`alarm.created`、`alarm.updated`、`alarm.resolved`、`command.updated`、`system.notice`。

## 订阅

```json
{
  "type": "subscription.set",
  "request_id": "client-uuid",
  "data": {
    "device_ids": ["device-001"],
    "channels": ["telemetry", "alarm", "command"]
  }
}
```

服务器必须逐个校验 Workspace 和权限。

## 连接模型

```text
Business Event
→ Bounded Queue
→ Coalescing/Priority
→ One Application Writer
→ WebSocket
```

单 Writer 是平台架构约束，用于顺序、超时、背压和关闭管理。

## 队列策略

- telemetry：按设备合并；
- state：有界 FIFO；
- alarm：持久化后通知；
- command：可由 REST/Event Replay 恢复；
- 持续慢客户端：降频，仍满则关闭。

## 心跳与恢复

- 协议 Ping/Pong；
- 读写 deadline；
- 指数退避重连；
- `GET /devices/{id}/state` 获取快照；
- `GET /events?after=cursor` 补缺失事件；
- Cursor 过期时重新获取完整快照。
