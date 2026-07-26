# ADR 0004: Transactional Outbox

- 状态：Accepted
- 日期：2026-07-26

## 背景

Command DB 事务与 MQTT 发布无法原子提交。

## 决策

同事务写 Command + Outbox，后台 Publisher 发布，Reply 消费幂等。

## 结果

获得可恢复最终一致性；增加 Outbox 清理、重试和监控。

## 备选

DB 后直接 Publish 或分布式事务；不采纳。
