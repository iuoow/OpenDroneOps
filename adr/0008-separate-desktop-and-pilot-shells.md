# ADR 0008: Separate Desktop and Pilot Shells

- 状态：Accepted
- 日期：2026-07-26

## 背景

DJI Pilot 2 WebView 是触控、户外、紧凑环境，桌面运营台则需要地图、表格、审计和多面板。直接响应式压缩会导致信息密度和交互目标冲突。

## 决策

Desktop App Shell 与 Pilot Shell 使用独立入口、导航和页面范围，共享领域契约、Design Tokens、API Client 和基础组件。

## 结果

增加少量 Shell 维护成本，但现场和桌面体验更清晰；JSBridge 通过 Adapter 隔离。

## 备选

一套页面完全响应式复用；不采纳。
