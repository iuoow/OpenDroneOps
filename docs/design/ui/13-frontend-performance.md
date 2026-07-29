# 前端性能

## Core Web Vitals

真实用户第 75 百分位目标：

- LCP ≤ 2.5s；
- INP ≤ 200ms；
- CLS ≤ 0.1。

## 业务指标

| 指标 | 目标 |
|---|---|
| App Shell 可交互 | <2s 开发基准 |
| 地图首次可见 | <3s |
| 设备选择到面板更新 P95 | <200ms |
| WS 消息到 UI P95 | <250ms |
| 地图视觉更新 | 2～5Hz |
| 表格 10k 记录 | 虚拟滚动 |

## Bundle

- 路由级拆包；
- 地图库懒加载；
- ECharts 按需导入；
- Pilot Shell 独立入口；
- 首屏不加载轨迹和运维模块。

初始预算建议：

```text
App Shell JS gzip        <= 250KB
Overview extra JS gzip   <= 350KB
Pilot Shell JS gzip      <= 200KB
```

## 实时更新

```text
WebSocket
→ event buffer
→ normalize
→ store update
→ requestAnimationFrame batch
→ component selectors
```

禁止每条遥测触发全局 Store 全量重算。

## 地图与表格

GeoJSON clustering、当前视野、轨迹简化、批量更新和 Worker；表格使用服务端 Cursor、虚拟滚动和稳定 Row Key。

## 测量

## Automated bundle gate

Task 39 turns the two independently deployable initial JavaScript budgets into
CI checks. `npm run build` emits the Vite manifest, then `npm run
check:bundle` follows each entry's static `imports` recursively and sums the
gzip bytes of the resulting JavaScript chunks.

- Operations entry (`index.html`): at most 250 KiB gzip.
- Pilot entry (`pilot.html`): at most 200 KiB gzip.

Dynamic imports are intentionally excluded: they are not initial-load costs and
must be assessed with the route or feature that requests them. The historical
"Overview extra" guidance remains a design constraint until Overview-specific
assets are split into a separately measurable dynamic entry. The gate itself
is covered by `npm run test:bundle` and runs in CI after every production build.

Lighthouse 只作实验室参考；使用 `web-vitals` 上报真实用户，并按桌面/Pilot、网络、设备和规模拆分。性能回归进入 CI。
