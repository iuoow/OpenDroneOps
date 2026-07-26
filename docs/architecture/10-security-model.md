# 安全模型

## Implemented HTTP baseline

The API applies `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
`Referrer-Policy: no-referrer` and `Cache-Control: no-store` to every response.
Request bodies are capped at 1 MiB at the HTTP boundary. Production startup
requires `mqtts://` and a loopback-only management listener; TLS/ACL material
remains deployment-owned and is never committed.

## 威胁

MQTT 凭证泄露、设备伪造、Topic 越权、消息重放、租户越权、危险指令、WebSocket 越权、日志泄露、供应链和默认配置风险。

## 设备接入

生产要求：TLS、独立身份、Broker ACL、禁止匿名、凭证轮换、连接限速、Client ID 冲突检测。Serial Number 不是认证凭证。

## 指令

- 危险命令默认关闭；
- Method 白名单和风险级；
- 权限、状态前置条件、过期时间和审计；
- 高风险二次确认；
- 不自动重试高风险命令；
- 紧急禁用开关。

## WebSocket

握手鉴权、Origin 策略、连接上限、频道级权限、Token 过期、Payload 限制、读写超时。

## 日志与原始数据

禁止记录 APP Key/License、MQTT 密码、Authorization、JWT Secret、TLS 私钥。高频 Payload 日志只记摘要；原文保留可配置、可脱敏、有审计。

## 供应链

版本固定、SBOM、漏洞扫描、License 清单、更新经测试合并、不从非官方 Fork 引入关键协议依赖。
