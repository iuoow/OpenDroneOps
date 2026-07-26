# 导航与布局

## Desktop App Shell

```text
┌─────────────────────────────────────────────────────────────┐
│ Logo  Workspace ▼  Search       Realtime ●  Alerts  User   │ 56
├──────────┬──────────────────────────────────────────────────┤
│ Overview │                                                  │
│ Devices  │                  Page Content                    │
│ Alarms   │                                                  │
│ Commands │                                                  │
│ Replay   │                                                  │
│ Ops      │                                                  │
└──────────┴──────────────────────────────────────────────────┘
    72/220
```

### 侧栏

- 折叠态 72px，展开态 220px；
- 主内容不可被覆盖；
- 图标有 Tooltip 和可访问名称；
- 导航不超过两层。

### 顶栏

稳定显示 Workspace、全局搜索、实时连接、严重告警和用户菜单。

### Context Panel

- 默认 360px；
- 可调整 320～520px；
- 普通设备详情不用全局 Modal；
- 小屏转为全屏页；
- 打开时更新 URL Query。

## 响应式档位

| 档位 | 宽度 | 行为 |
|---|---:|---|
| Wide Desktop | ≥1440 | 展开导航、地图和详情并排 |
| Desktop | 1200～1439 | 折叠导航、详情 360px |
| Compact Desktop | 1024～1199 | 详情 Drawer |
| Tablet/Pilot | 768～1023 | 独立紧凑布局 |
| Mobile | <768 | 仅有限状态和告警功能 |

## Pilot Shell

```text
┌──────────────────────────────────────────┐
│ Workspace  Cloud ●  Device ●   15:42    │ 48
├──────────────────────────────────────────┤
│             Current Task / Map           │
├──────────────────────────────────────────┤
│ Home   Device   Alerts   Work   More     │ 64
└──────────────────────────────────────────┘
```

Pilot 端使用底部导航、至少 44px 触控目标、不依赖 Hover，并通过 `PilotBridgeAdapter` 隔离 JSBridge。

## Z-index

```text
base 0
sticky header 100
map controls 200
drawer 300
popover 400
toast 500
modal 600
critical overlay 700
```
