# 领域模型

## 聚合

### Workspace

`id`、`name`、`status`、`created_at`。

### Device

`id`、`workspace_id`、`vendor`、`serial_number`、`gateway_serial_number`、`product_model`、`device_type`、`capabilities`、`status`。

唯一约束：`(workspace_id, vendor, serial_number)`。

### DeviceRelationship

Gateway 与子设备：`parent_device_id`、`child_device_id`、`relationship_type`、有效期。

### DeviceLatestState

`state_version`、设备/服务端时间、在线、经纬度、高度、电量、模式和扩展 Payload。

### DeviceEvent

不可变事件：`event_id`、设备、网关、类型、Method、时间、序号、Raw Message 和 Payload。

### Alarm

`dedup_key`、类型、级别、状态、首次/最近时间、次数、确认、恢复。

### Command

内部 `id`、目标设备、网关、Method、状态、风险级、幂等 Key、请求摘要、`dji_tid/dji_bid`、参数、有效期和结果。

### CommandEvent / OutboxEvent / RawMessage

分别用于不可变状态历史、可靠发布和协议诊断。

## 不变量

- Workspace 内设备序列号唯一；
- Command 终态不可回退；
- 同 Idempotency Key 的不同请求摘要拒绝；
- 活动 Alarm 同 dedup key 唯一；
- Latest State 只接受更高版本；
- Outbox 与业务变更同事务。
