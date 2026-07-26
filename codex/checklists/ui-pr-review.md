# UI PR Review

## UX
- [ ] 在线、实时连接和数据新鲜度分开。
- [ ] 异步操作有过程。
- [ ] 空、错、加载、Stale 分开。
- [ ] 危险动作明确对象和后果。
- [ ] Toast 不是唯一反馈。

## Design System
- [ ] 使用 Token。
- [ ] 无新硬编码状态色、间距和 z-index。
- [ ] Light/Dark 正常。
- [ ] 状态不只靠颜色。
- [ ] 支持减少动态。

## Realtime/Map
- [ ] 高频更新批量渲染。
- [ ] 无海量 HTML Marker。
- [ ] 断线 Snapshot + Cursor。
- [ ] 历史/实时明显。

## Accessibility
- [ ] 键盘、焦点、对比度和尺寸通过。
- [ ] 地图有等价列表。
- [ ] 动态播报不过量。

## Performance
- [ ] 路由拆包、虚拟表格、Bundle Budget、CWV 和无泄漏。

## Pilot
- [ ] 独立 Shell、Bridge Adapter、44px 触控、无 Hover、无 DRC。
