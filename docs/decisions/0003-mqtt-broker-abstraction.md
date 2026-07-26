# ADR 0003: MQTT Broker Abstraction

- 状态：Accepted
- 日期：2026-07-26

## 背景

希望默认易启动且兼容 EMQX/HiveMQ；EMQX 5.9+ 有 BSL 限制。

## 决策

默认 Mosquitto，应用只依赖 MQTT 标准，提供可选 Broker Profile。

## 结果

降低许可证和部署门槛，减少 Broker 专有能力绑定。

## 备选

仅支持 EMQX或自研 Broker；不采纳。
