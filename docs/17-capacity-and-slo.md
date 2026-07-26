# 容量与 SLO

## 基准

1000 设备 × 1 msg/s × 1KB × 1 小时；1000 WS 连接；一个热点设备 100 msg/s。

## MVP 目标

| 指标 | 目标 |
|---|---|
| 遥测到可查询 P95 | < 500ms |
| 遥测到 WS 入队 P95 | < 500ms |
| 告警 P99 | < 2s |
| 关键事件持久化 | > 99.9% |
| 无界队列 | 0 |
| 持续 goroutine 泄漏 | 0 |
| 单设备异常 | 可隔离 |

## 容量公式

`messages/s = online_devices × per_device_rate`  
`ingress_bytes/s = messages/s × average_payload`  
`daily_raw = ingress_bytes/s × 86400`

## 降级

优先保证安全告警、指令结果、在线状态、状态变化，再保证普通 OSD 和历史统计。可合并 OSD、降低 WS 频率、限制历史查询和异常租户。
