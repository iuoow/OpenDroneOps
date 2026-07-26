# 可观测性

## 关联字段

`trace_id`、`request_id`、`workspace_id`、`device_id`、脱敏 gateway SN、`command_id`、`tid`、`bid`、Topic Kind、`event_id`。

## MQTT 指标

连接、重连、消息数/字节、解码错误、重复、旧状态、处理耗时、队列深度、合并/丢弃。

## WebSocket 指标

连接数、建立/断开原因、发送队列、合并、丢弃、慢客户端、写耗时。

## Command/Outbox 指标

命令创建、状态、端到端耗时、超时、孤儿 Reply、Outbox Pending 和重试。

## Trace

`mqtt.receive` → `dji.decode` → `deduplicate` → `state.persist` → `alarm.evaluate` → `websocket.enqueue`；指令链路增加 `command.create`、`outbox.publish`、`mqtt.command_reply`。

## 日志

JSON 结构化；高频 OSD 不逐条 INFO；采样；Recover 记录堆栈但不泄露秘密。pprof 仅在受保护管理端口。
