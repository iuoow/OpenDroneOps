# DJI Cloud API 集成

## 事实源

DJI 外部协议以当前官方 Cloud API 文档为准。停止维护的官方 Demo 只可帮助理解流程，不能作为安全或生产架构基线。

## Topic

| Topic | 方向 | 用途 |
|---|---|---|
| `thing/product/{device_sn}/osd` | 设备→云 | 定频属性 |
| `thing/product/{device_sn}/state` | 设备→云 | 变化属性 |
| `thing/product/{gateway_sn}/services` | 云→设备 | 服务/指令 |
| `thing/product/{gateway_sn}/services_reply` | 设备→云 | 服务结果 |
| `thing/product/{gateway_sn}/events` | 设备→云 | 事件/进度 |
| `thing/product/{gateway_sn}/events_reply` | 云→设备 | Event ACK |
| `thing/product/{gateway_sn}/requests` | 设备→云 | 设备请求 |
| `thing/product/{gateway_sn}/requests_reply` | 云→设备 | 请求回复 |
| `sys/product/{gateway_sn}/status` | 设备→云 | 在线/拓扑状态 |
| DRC Topics | 双向 | 独立实时控制，MVP 不实现 |

## Envelope

常见字段：

- `tid`
- `bid`
- `timestamp`
- `gateway`
- `method`
- `need_reply`
- `seq`
- `data`

Adapter 必须保留未知字段、原始 Topic、Payload 哈希和服务端接收时间。

## OSD 与 State

- OSD 更新最新状态和轨迹采样；
- State 产生状态变化事件；
- 旧消息不能覆盖新版本；
- 同时保存设备时间和服务端时间；
- 设备重启导致序号重置时应建立新的 session epoch。

## Services / Events

- Broker ACK 不等于设备完成；
- 通过 `tid/bid/method` 关联 Reply 和进度 Event；
- `need_reply` 事件需要按协议及时回复；
- 不同 Method 的完成条件由 Method Registry 配置；
- 未知 Method 进入诊断/隔离，不导致 Worker 退出。

## Method Registry

```go
type MethodDefinition struct {
    Name             string
    Direction        Direction
    Capability       string
    ReplyMode        ReplyMode
    ProgressEvent    string
    Timeout          time.Duration
    RiskLevel        RiskLevel
}
```

MVP 只注册模拟 LOW 风险 Method。

## DRC

DRC 有独立 Topic、顺序和低延迟要求。MVP 不订阅、不发布、不预留可被误用的危险入口。
