# PR Review

## Protocol
- [ ] device_sn/gateway_sn 正确。
- [ ] tid/bid/method 保留。
- [ ] QoS 1 重复幂等。
- [ ] 未知字段/Method 不崩溃。
- [ ] MQTT ACK 未当作业务成功。
- [ ] need_reply 已处理。

## Concurrency
- [ ] 无无限 goroutine/队列。
- [ ] 关闭安全。
- [ ] 锁内无网络 I/O。
- [ ] race test 覆盖。

## Data/Security
- [ ] 唯一约束支持幂等。
- [ ] 外部调用不在事务中。
- [ ] Redis 可重建。
- [ ] Workspace 条件完整。
- [ ] 日志脱敏、权限服务端验证。
- [ ] 危险 Method 默认关闭。

## Operations
- [ ] 指标、日志、Trace。
- [ ] 优雅退出。
- [ ] 配置校验。
- [ ] 版本固定。
- [ ] 回滚/兼容策略明确。
