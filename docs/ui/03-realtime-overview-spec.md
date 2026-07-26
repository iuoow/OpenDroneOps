# 实时态势页规格

## 目标

在 5 秒内帮助操作员确定严重告警设备、选中设备、实时链路状态、数据是否过期，以及最近关键事件。

## 页面结构

```text
┌──────────────── Top Bar ───────────────────────────────────┐
│ Workspace │ 12/14 Online │ 2 Critical │ Realtime Connected │
├──────┬───────────────────────────────┬──────────────────────┤
│ Nav  │             Map               │ Device Context       │
│      │                               │ Freshness/Telemetry  │
│      │                               │ Alarms/Safe Actions  │
├──────┴───────────────────────────────┴──────────────────────┤
│ Event Timeline                                             │
└─────────────────────────────────────────────────────────────┘
```

## 默认地图内容

显示 Dock、Aircraft、活跃严重告警和选中设备最近轨迹尾迹。默认不显示全部历史轨迹、所有普通事件、复杂天气和运维调试信息。

## 顶部摘要

只展示可行动信息：在线设备、严重告警、实时连接和消息延迟。避免虚荣 KPI。

## Device Context Panel

### Header

设备名称、型号、Online/Offline/Stale、最后更新时间、定位和完整详情。

### 状态区

电量、高度、速度、航向、模式、信号和所属 Dock。

### 告警区

仅活动告警，按严重程度排序。

### 操作区

MVP：刷新状态、打开轨迹、查看事件和触发模拟告警。

## 实时事件时间线

默认只显示 Online/Offline、State Change、Alarm、Command Progress；高频 Telemetry 不逐条显示。

## 数据新鲜度

```text
Fresh       <= 2 × expected interval
Delayed     <= 5 × expected interval
Stale       > 5 × expected interval
Offline     broker status or business timeout
```

示例：

```text
在线 · 2 秒前更新
数据延迟 · 14 秒前更新
数据可能已过期 · 48 秒未更新
设备离线 · 最后上报 4 分钟前
```
