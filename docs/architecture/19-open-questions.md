# 前置问题决策结果

Task 0 已根据“开发前自动裁决”的授权完成默认决策。后续如需改变，必须新增 ADR
并同步依赖、契约和验收标准。

- [x] 项目名称：`OpenDroneOps`；预期 GitHub：`github.com/iuoow/OpenDroneOps`
- [x] 许可证：Apache-2.0
- [x] 固定版本：见 `docs/development/dependencies.md`
- [x] 数据库迁移：Goose + 显式 SQL
- [x] Redis Go Client：`github.com/redis/go-redis/v9`
- [x] MVP 登录：不实现真实登录；保留 Workspace 边界和鉴权接口
- [x] 租户模型：后端多 Workspace，Desktop UI 单 Workspace
- [x] 原始 OSD：正常 7 天或采样；错误/未知 30 天；命令/审计 1 年
- [x] Pilot 2 WebView：规格保留，MVP 后实现
- [x] 首批设备：仅模拟 Dock/Aircraft；仅 `sim_status_refresh` LOW 风险命令
- [x] 真实设备测试：MVP 不执行，后续建立授权 SOP
- [x] 生产环境：Task 0 不假设云厂商；先完成本地 Compose
- [x] 监控平台：OpenTelemetry + Prometheus-compatible 指标；外部平台后置

仍需在真实 DJI 接入前单独确认：

- 首批真实型号及当前官方 Product Supported 范围；
- DJI 凭证申请、测试授权和生产安全 SOP；
- 生产部署形态、备份策略和监控落地平台。
