# 模拟器规格

## 模式

- DJI-Compatible：真实 Topic 与 Envelope，用于 E2E；
- Normalized：直接领域事件，仅单测。

## 能力

Gateway + Aircraft 拓扑；上线/离线；OSD；State；Events/Reply；Services/Reply；进度、失败和超时；设备重启与序号重置。

## 故障

重复、乱序、延迟、丢弃、非法 JSON、未知 Method、错误 SN、大 Payload、频繁重连、热点设备、重连风暴。

## 可重复

每个场景有固定 seed。相同版本、配置和 seed 产生相同事件序列。

## 录制

包含相对时间、Topic、QoS、Retain、Payload、方向。真实录制必须脱敏且不进入公共仓库。
