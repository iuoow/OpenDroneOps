# 实时地图规格

## 技术

MapLibre GL JS。大批量设备使用 WebGL 图层，不为每个设备创建独立 HTML Marker。

## 标记编码

| 维度 | 视觉 |
|---|---|
| 设备类型 | 图标或形状 |
| 在线状态 | 填充颜色 |
| 告警 | 外环和徽标 |
| 航向 | 图标旋转 |
| 数据过期 | 透明度和时钟标记 |
| 选中 | 高对比描边和 Halo |

状态不得只依赖颜色。

## 图层

```text
base-map
geofence-fill
mission-area
trajectory-history
trajectory-tail
device-clusters
device-symbols
device-alert-rings
selected-device
labels
```

## 聚合

- 低缩放 Cluster；
- 中缩放按 Dock 或设备组聚合；
- 高缩放单设备；
- 选中设备始终最上层；
- Cluster 点击放大并在侧栏列出严重设备。

## 地图与列表

- 地图点击打开详情；
- 列表点击定位；
- Hover 只高亮，不改变选中；
- 清除选择保留视野；
- URL 保存中心、缩放和选中；
- 浏览器返回恢复视图。

## 实时位置

- 接收频率与视觉刷新分离；
- UI 刷新 2～5Hz；
- 短距离插值；
- 大跳点直接移动并记录异常；
- 离线停止插值；
- 后台标签页降频；
- 当前视野和关注设备优先。

## 大数据

- GeoJSON Source + clustering；
- 轨迹按时间或视野分块；
- 按缩放级别简化；
- 批量更新；
- 避免频繁替换完整 GeoJSON；
- 监控帧率、Source 更新时间和长任务。

## 无障碍

地图外提供设备列表、告警列表、搜索、定位按钮和当前视野摘要。地图 Pin 可使用 Target Size 的 Essential 例外，但同一功能必须有更易操作的等价控件。
