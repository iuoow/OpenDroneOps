# 信息架构

## 一级模块

```text
实时态势
设备管理
告警中心
指令中心
轨迹回放
系统运行
```

## 路由草案

```text
/login
/workspaces
/app/:workspaceId/overview
/app/:workspaceId/devices
/app/:workspaceId/devices/:deviceId
/app/:workspaceId/alarms
/app/:workspaceId/alarms/:alarmId
/app/:workspaceId/commands
/app/:workspaceId/commands/:commandId
/app/:workspaceId/replay/:deviceId
/app/:workspaceId/operations
/app/:workspaceId/quarantine
/pilot/:workspaceId/home
/pilot/:workspaceId/device/:deviceId
/pilot/:workspaceId/alarms
/pilot/:workspaceId/diagnostics
```

## 模块职责

### 实时态势

回答“现在发生了什么”：实时地图、设备状态、活跃告警、最近事件、快捷定位和实时连接。

### 设备管理

回答“有哪些设备、它们是什么”：设备列表、拓扑、型号、能力、最新状态和历史入口。

### 告警中心

回答“哪些问题需要处理”：未处理、已确认、已恢复、严重程度、处置时间线和关联对象。

### 指令中心

回答“谁对什么设备做了什么、结果如何”：指令状态、参数、前置检查、`command_id/tid/bid` 和审计。

### 轨迹回放

回答“过去某个时段发生了什么”：地图轨迹、时间轴、高度、电量、速度、告警和指令事件。

### 系统运行

面向开发者和运维：MQTT、消息积压、WebSocket、Outbox、数据库、Redis、未知协议和隔离消息。

## 导航规则

- 当前 Workspace 始终可见；
- 主导航稳定，不因地图操作变化；
- 详情使用 URL 表达，可刷新和分享；
- 浏览器返回恢复地图位置、筛选和选中对象；
- 高级运维模块按权限隐藏；
- Pilot Shell 使用独立导航。
