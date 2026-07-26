# 测试策略

## 层次

1. 单元：Topic、状态机、去重、规则；
2. 组件：Repository、Redis、WebSocket；
3. 契约：DJI Envelope、OpenAPI、AsyncAPI；
4. 集成：Mosquitto + PostgreSQL + Redis；
5. E2E：Simulator → MQTT → API/WS；
6. 负载和故障。

## 必测

OSD、State、非法 Topic/JSON、未知字段/Method、QoS 1 重复、乱序、设备时间回退、Gateway/Child SN、`need_reply`、Services Reply、孤儿 Reply、命令超时。

## 并发

`go test -race ./...`、反复连接关闭、Worker Shutdown、队列满、慢客户端、DB/Redis 超时、同命令并发重复。

## 原则

不用任意 sleep；使用 fake clock；固定随机 seed；自动清理；不依赖真实密钥；容器版本固定。

## 故障注入

Broker 重启、网络延迟/丢包、DB 中断、Redis 清空、Outbox 发布失败、WS 客户端停止读取、DB 提交后 Publisher 退出、处理成功但确认前退出。
