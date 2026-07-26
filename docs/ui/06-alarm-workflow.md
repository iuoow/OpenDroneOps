# 告警交互与工作流

## 状态

```text
OPEN → ACKNOWLEDGED → RESOLVED
```

确认只表示有人接手，不代表问题消失。

## 页面

```text
┌──────── Alarm Filters / Summary ──────────────────────────┐
├───────────────────┬────────────────────────────────────────┤
│ Alarm Queue       │ Alarm Detail                           │
│ Severity/Device   │ Explanation / Device Context           │
│ Duration/Count    │ Related Events / Handling Timeline     │
└───────────────────┴────────────────────────────────────────┘
```

## 新告警行为

- 地图脉冲一次；
- 顶部计数更新；
- 队列插入；
- Critical 可播放可配置提示音；
- 不抢夺输入焦点；
- 同 dedup key 更新次数；
- Toast 不是唯一承载。

## 严重程度

```text
INFO      信息
WARNING   需要关注
CRITICAL  立即处置
```

失败、离线和过期是状态，不应全部映射成 Critical。

## 详情

告警解释、设备位置、当前状态、数据新鲜度、首次/最近时间、重复次数、相关事件、推荐检查、备注、关联指令和审计。

## 操作

Acknowledge 文案使用“确认接手此告警”，记录操作者和时间线，不隐藏告警。

Resolve 默认由规则自动恢复；人工解决需要原因、备注、权限和风险提示。
