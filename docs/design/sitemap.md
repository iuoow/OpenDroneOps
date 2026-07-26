# Sitemap

```mermaid
flowchart TD
    Login --> Workspace
    Workspace --> Overview
    Overview --> DeviceDetail
    Overview --> AlarmDetail
    Overview --> CommandDetail
    Overview --> Replay
    Workspace --> Devices
    Devices --> DeviceDetail
    Workspace --> Alarms
    Alarms --> AlarmDetail
    Workspace --> Commands
    Commands --> CommandDetail
    Workspace --> Operations
    Operations --> Quarantine

    PilotLogin --> PilotHome
    PilotHome --> PilotDevice
    PilotHome --> PilotAlerts
    PilotHome --> PilotWork
    PilotHome --> PilotDiagnostics
```
