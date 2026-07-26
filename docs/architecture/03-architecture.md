# 架构设计

```mermaid
flowchart TB
    MQTT[MQTT Broker]
    Ingest[MQTT Ingestion Worker]
    Adapter[DJI Protocol Adapter]
    Bus[In-process Domain Event Bus]
    Twin[Device Twin]
    Alarm[Alarm]
    Cmd[Command]
    Outbox[Outbox Publisher]
    DB[(PostgreSQL)]
    Redis[(Redis)]
    API[REST API]
    WS[WebSocket Hub]
    UI[Vue Console]
    Sim[Simulator]

    Sim <--> MQTT
    MQTT --> Ingest
    Ingest --> Adapter
    Adapter --> Bus
    Bus --> Twin
    Bus --> Alarm
    Bus --> Cmd
    Twin --> DB
    Twin --> Redis
    Alarm --> DB
    Cmd --> DB
    DB --> Outbox
    Outbox --> MQTT
    API --> Twin
    API --> Alarm
    API --> Cmd
    Bus --> WS
    WS --> UI
    UI --> API
```

## 部署单元

- `server`：REST、WebSocket 和领域模块；
- `worker`：MQTT 消费和 Outbox 发布；
- `simulator`：DJI 协议模拟。

## 上行

```text
MQTT
→ Topic Parser
→ Envelope Decode
→ Validation
→ Deduplication
→ Domain Event
→ State/Event Persistence
→ Alarm
→ WebSocket
```

## 下行

```text
REST Command
→ Authorization / Preconditions
→ Command + Outbox Transaction
→ MQTT Services
→ Services Reply / Events
→ Command Transition
→ WebSocket Notification
```

## 边界

- Domain 不导入 DJI DTO；
- Repository 不向 Handler 暴露数据库模型；
- Redis 失败不能破坏关键 DB 事务；
- 第一版 Event Bus 为进程内接口，后续可替换；
- 不在 DB 事务内调用 MQTT/HTTP。
