# UI/UX Codex 任务

> 后端快照和 WebSocket 契约稳定后执行；前期可使用 Mock Server。

## UI Task 0 — 技术基线

核验 Vue、Vite、TypeScript、Router、Pinia、MapLibre、ECharts、测试和可访问性工具，固定版本和许可证，建立 Bundle Budget，不实现业务页面。

## UI Task 1 — Design System

导入 Tokens，完成主题、排版、Button/Input/Badge/Notice/Drawer/Modal、状态组件和组件预览。

## UI Task 2 — App Shell

Desktop Shell、Workspace、Navigation、连接状态、URL 状态、Error Boundary 和响应式布局。

## UI Task 3 — Overview Mock

地图、设备图层、Context Panel、事件时间线，以及正常、空、加载、错误、Stale、Disconnected 状态。

## UI Task 4 — Snapshot + WebSocket

REST 快照、WS 增量、Cursor、断线恢复、批量渲染和版本合并。

## UI Task 5 — Devices

搜索、筛选、保存视图、虚拟表格、详情、拓扑和事件入口。

## UI Task 6 — Alarms

队列、详情、Acknowledge、时间线和去重更新体验。

## UI Task 7 — Commands

允许 Method、前置检查、确认、进度、未知结果和审计。

## UI Task 8 — Replay

轨迹、时间轴、图表、事件同步、抽样和 Worker。

## UI Task 9 — Operations

连接、队列、Outbox、隔离和外部监控跳转。

## UI Task 10 — Pilot Shell（MVP 后）

独立入口、PilotBridgeAdapter、Mock Bridge、紧凑页面、触控和诊断日志。
该任务保留规格和 Mock 设计，但不进入 MVP 主线；只有 Task 0～9 和 Quality Gate
完成后才排期。

## UI Task 11 — Quality Gate

WCAG 2.2 AA、Playwright、axe、键盘、Core Web Vitals、1000 设备负载、Bundle Budget 和内存检查。
