# Codex 启动说明

## 首次指令

```text
先阅读 AGENTS.md、README.md、CODEX_TASKS.md、docs/、adr/ 和 codex/checklists/。
不要编写业务代码。
Task 0 的默认决策已完成：
1. 依赖和许可证见 `DEPENDENCIES.md`；
2. 项目决策见 `adr/0011-task-0-baseline-decisions.md`；
3. API、AsyncAPI、JSON Schema 和 SQL 已完成第一轮对齐；
4. 未完成的真实 DJI 凭证、型号和生产 SOP 仍不得进入实现；
5. Task 0、Task 1、Task 2、Task 3、Task 4 已完成；下一步只能执行 Task 5，不得跨越任务边界。
```

## 会话策略

- 每次只做一个 Task。
- 新会话先让 Codex 总结已加载的项目规则和当前验收条件。
- Task 0 单独完成；协议、模拟器、MQTT Worker 按顺序合并。
- 每次结束按以下格式汇报：

```text
完成内容：
修改文件：
运行命令：
测试结果：
架构决策：
安全影响：
性能影响：
已知限制：
下一任务建议：
```

## 必须人工决定

- 项目最终名称和仓库；
- Apache-2.0 或 MIT；
- 固定依赖版本；
- 数据库迁移工具；
- MVP 是否实现用户登录；
- 是否兼容 DJI Pilot 2 WebView；
- 原始 MQTT Payload 保留策略；
- 首批真实设备型号与安全命令；
- 真实设备测试 SOP；
- 生产部署形态。

## 人工审查重点

- 是否超出 Task；
- 是否增加未经批准的依赖；
- 是否使用 `latest`；
- DJI DTO 是否泄漏到领域层；
- 是否把 MQTT ACK 当作执行成功；
- 是否存在无限队列、无限重试或 goroutine 泄漏；
- 是否覆盖重复、乱序、断线和慢客户端；
- 是否泄露秘密；
- 契约和文档是否同步更新。
