# ADR 0006: DJI Adapter Boundary

- 状态：Accepted
- 日期：2026-07-26

## 背景

DJI Topic、Envelope、型号和 Method 会演进，未来可能接入其他厂商。

## 决策

DJI DTO、Topic Parser 和 Method Registry 仅位于 Adapter；Domain 只接收标准事件。

## 结果

协议升级影响受控，但需要显式映射和契约测试。

## 备选

业务层直接使用 DJI JSON；不采纳。
