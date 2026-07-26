# 数据库设计

## 原则

- PostgreSQL 是事实源；
- 原始消息、不可变事件和最新状态分表；
- 唯一约束负责最终幂等；
- 查询带 `workspace_id`；
- 深分页使用稳定 Cursor；
- 最新状态使用版本条件更新；
- 外部发布使用 Outbox。

## 事务

设备事件事务：

1. Raw/Processed Message；
2. Device Event；
3. Latest State；
4. 必要的 Outbox/Domain Event。

指令事务：

1. Command；
2. Command Event；
3. Outbox Event。

事务中禁止 MQTT/HTTP。

## 数据保留初始建议

| 数据 | 建议 |
|---|---|
| 最新状态 | 随设备保留 |
| 设备事件 | 90 天 |
| 轨迹 | 30 天 |
| 正常 OSD 原文 | 7 天或采样 |
| 错误/未知消息 | 30 天 |
| 指令/审计 | 1 年或合规要求 |
| WS Replay Event | 24 小时～7 天 |

达到明显规模后再评估分区、PostGIS、TimescaleDB、ClickHouse 或 Parquet。
