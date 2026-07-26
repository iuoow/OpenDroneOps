# 开源许可证决策

项目许可证已自动确定为 Apache-2.0，并已添加根目录 `LICENSE`。

选择原因：

- 对企业采用、专利授权和衍生作品分发边界更清晰；
- 与 Go、Vue、MapLibre 及大多数基础依赖兼容；
- 适合作为协议适配、可靠性和 IoT 工程参考项目；
- 不把项目本身绑定到 Redis 7.4+/8.x 的 source-available 条款。

第三方注意：

- EMQX 5.9+ 使用 BSL 1.1，集群、托管和嵌入式商业场景存在限制。
- Mosquitto、HiveMQ CE 和 Go 依赖均有各自许可证。
- 不复制停止维护的 DJI Demo 代码和安全设计。
- 本文件不是法律意见。第三方依赖仍以各自许可证为准，完整清单见 `docs/development/dependencies.md`。
