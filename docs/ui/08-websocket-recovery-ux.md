# 实时连接与恢复体验

## 状态分层

1. 浏览器到平台 WebSocket；
2. 平台到 MQTT Broker；
3. 设备最后遥测时间。

示例：

```text
实时连接：正常
平台设备通道：正常
设备遥测：2 秒前
```

## 顶部状态

```text
● 实时连接正常
◐ 正在重连（第 2 次）
○ 实时连接已断开
```

## 断线

1. 保留数据；
2. 标记可能过期；
3. 禁止依赖实时状态的危险操作；
4. 不弹阻塞 Modal；
5. 指数退避重连；
6. 显示最近成功同步时间。

## 恢复

```text
Connected
→ Fetch Snapshot
→ Fetch Events After Cursor
→ Reconcile by Version
→ Resume Incremental
```

文案：

```text
正在恢复实时数据…
已连接，正在补齐 12 条事件
实时数据已恢复
```

Cursor 过期时重新加载完整状态。Snapshot 失败时保留旧数据并显示 Request ID。

## 多标签页

可用 BroadcastChannel 协调，限制同用户连接数，后台降频，避免重复播放相同告警声音。
