# 设计系统

## 视觉方向

```text
工业控制
地理信息
高信息密度
清晰克制
稳定专业
```

避免赛博朋克、大面积玻璃拟态、复杂渐变、发光边框和装饰性仪表盘。

## 主题

Light、Dark、System，以及可选 Pilot Outdoor High Contrast。

## 字体

系统字体优先。编号、时间和遥测数字使用等宽数字：

```css
font-variant-numeric: tabular-nums;
```

## 字号

| Token | Size | 用途 |
|---|---:|---|
| xs | 12 | 地图标签、辅助 |
| sm | 13 | 紧凑表格 |
| md | 14 | 默认正文 |
| lg | 16 | 强调正文 |
| xl | 20 | 页面标题 |
| 2xl | 24 | 关键标题 |
| 3xl | 32 | 少量关键状态 |

## 约束

- 4px 间距基线；
- 圆角仅 4、8、12 和 Pill；
- 阴影仅 Popup、Drawer、Modal 三档；
- 普通 Card 优先边框；
- 图标统一线性风格；
- 关键操作带文字。

## 核心组件

`StatusBadge`、`FreshnessIndicator`、`ConnectionIndicator`、`SeverityBadge`、`ProgressStepper`、`EmptyState`、`ErrorState`、`Skeleton`、`InlineNotice`。

## 动效

120ms、200ms、400ms 三档；支持 `prefers-reduced-motion`，Critical 告警不永久闪烁。
