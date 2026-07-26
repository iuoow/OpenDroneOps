# 部署蓝图

## 本地

PostgreSQL、Redis、Mosquitto、server、worker、simulator、可选 web。

## 容器

多阶段构建、非 root、固定镜像版本/digest、SIGTERM、健康检查、stdout 日志、只读文件系统优先、资源限制。

## 生产

可从 2 个 server、1～N 个 worker、Broker、PostgreSQL HA、Redis、负载均衡开始。WebSocket 多实例前需完成跨实例事件路由；粘性会话不是最终可靠性方案。

## 升级

向后兼容迁移、先代码后启用字段、滚动发布、Worker 优雅停止、WS 客户端重连抖动，观察积压、重连和命令成功率。

## 恢复

PostgreSQL 备份/恢复演练；Redis 可重建；Command/Alarm 不依赖 Broker 内存；明确 RPO/RTO。
