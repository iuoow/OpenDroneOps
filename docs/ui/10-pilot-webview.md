# DJI Pilot 2 WebView UI

## 定位

Pilot 2 中运行的是开发者自定义 H5，并通过 JSBridge 与 Pilot 2 通信。Pilot Shell 不是桌面后台的缩小版。

## 范围

保留：当前任务、当前设备状态、现场告警、工单或备注、通信入口、诊断上传和云连接状态。

不放：大型设备表、组织权限、全局配置、复杂审计、多图表分析和完整轨迹回放。

## 交互

- 底部导航；
- 主触控目标至少 44×44 CSS px；
- 不依赖 Hover；
- 高对比户外模式；
- 字号不低于 14px；
- 关键按钮不贴屏幕边缘；
- 离线支持草稿和重试；
- 云连接和 Pilot 模块状态可见。

## JSBridge Adapter

```ts
interface PilotBridgeAdapter {
  isAvailable(): boolean
  verifyLicense(): Promise<LicenseResult>
  setWorkspace(workspaceId: string): Promise<void>
  configureApi(config: ApiConfig): Promise<void>
  configureWebSocket(config: WsConfig): Promise<void>
  getLogPath(): Promise<string | null>
  launchThirdPartyApp(uri: string): Promise<void>
}
```

浏览器环境使用 Mock Adapter，业务组件不得直接调用 `window.djiBridge`。

## 启动流程

```text
Load H5
→ Detect Bridge
→ Authenticate
→ Verify DJI License
→ Set Workspace
→ Configure API/WS
→ Load Required Pilot Module
→ Enter Pilot Home
```

## 诊断包

“上传 Pilot 诊断日志”需要用途和隐私说明、日志路径获取、加密上传、进度、诊断单号和审计；普通 UI 不暴露底层路径。
