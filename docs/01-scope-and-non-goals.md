# 范围与非目标

## MVP

- MQTT 设备接入；
- DJI OSD、State、Event、Reply 的通用适配；
- 设备主数据、拓扑、在线状态和最新状态；
- Redis 派生缓存；
- WebSocket 实时推送；
- 基础告警；
- 模拟安全指令和可靠状态机；
- 历史事件和轨迹；
- 本地 Docker Compose；
- 模拟器；
- 日志、指标、Trace、pprof。

## 非目标

- DRC 实时飞行控制；
- 自动起飞、降落、返航；
- 视频传输；
- DJI 全部型号和全部 Method；
- 完整 Pilot 2 H5；
- 复杂航线编排；
- AI、边缘计算；
- Kubernetes、Kafka/NATS、ClickHouse、Elasticsearch；
- 商业级 IAM；
- 对外托管 MQTT 服务。

## 非功能要求

- 网络调用有超时；
- 后台任务可取消；
- 队列有界；
- 重试有限；
- 命令、告警、审计可恢复；
- 状态消息可合并；
- 不记录秘密；
- 多租户查询带 Workspace 边界；
- 生产依赖版本固定。
