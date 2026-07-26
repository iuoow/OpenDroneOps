# MQTT 契约

## Broker

默认 Mosquitto，兼容验证 EMQX/HiveMQ CE。应用只依赖 MQTT 标准协议；Broker 专有 API 位于可选 Adapter。

## 连接

- 生产 TLS；
- 稳定 Client ID；
- 显式 Clean Start/Session Expiry/Keep Alive；
- 指数退避 + 抖动；
- 认证失败和网络失败分类；
- 连接、断开、重连指标；
- 优雅断开。

## QoS

| 数据 | 建议 |
|---|---:|
| 高频可覆盖状态 | 0 或 1 |
| 告警/重要事件 | 1 |
| 指令 | 1 |
| 明确要求的极少场景 | 2 |

QoS 1 至少一次，消费必须幂等。

## 去重

优先：

1. 协议稳定 ID；
2. `gateway + tid + bid + method + direction`；
3. Topic + Payload Hash + 时间窗口。

数据库唯一约束是最终防线；Redis 去重只是优化。

## 顺序

- 不追求全局有序；
- 按 `device_id`/`gateway_sn` 哈希分片；
- 单分片串行；
- sequence 缺失不能永久阻塞；
- 旧状态条件更新丢弃。

## 背压

- OSD：合并最新值；
- State：有界缓冲；
- Alarm/Event：持久化或隔离；
- Command Reply：高优先级；
- 每层记录容量、队列深度、丢弃/合并指标。

## 共享订阅

规模化阶段可使用 `$share/<group>/<filter>`，但仍需应用层设备分片，不能依赖 Broker 保证完整业务顺序。

## 错误分类

`malformed_topic`、`invalid_json`、`unsupported_method`、`unknown_device`、`duplicate`、`stale_state`、`repository_timeout`、`queue_full`、`ack_publish_failed`。
