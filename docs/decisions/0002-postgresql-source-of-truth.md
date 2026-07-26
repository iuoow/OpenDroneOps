# ADR 0002: PostgreSQL as Source of Truth

- 状态：Accepted
- 日期：2026-07-26

## 背景

设备、告警和命令需要事务、唯一约束和恢复。

## 决策

PostgreSQL 保存核心事实，Redis 只保存可重建派生状态。

## 结果

Redis 故障可降级；DB 压力通过索引、批量、分区和保留策略治理。

## 备选

Redis 主存储或多数据库起步；不采纳。
