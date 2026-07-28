# State Language v2

## Purpose

State language prevents ambiguous claims such as "online" or "live". The UI
must show the exact kind of evidence the operator is looking at. State tokens
are shared by desktop and Pilot, while each shell chooses its density.

## Independent state dimensions

| Dimension | Values | User-facing evidence | Must not be confused with |
|---|---|---|---|
| Browser realtime | Connected, recovering, disconnected | Connection label plus recovery phase | Device or MQTT presence |
| Device presence | Online, offline, unknown | Device-status icon and last known transition | Telemetry freshness |
| Telemetry freshness | Fresh, delayed, stale, unavailable | Last update time and threshold explanation | Device health |
| Health | Normal, attention, critical, unknown | Named condition and associated alert | Connection state |
| Task / command | Not applicable, queued, published, accepted, executing, succeeded, failed, timed out, unknown | Step, timestamp, and source | Business success from MQTT acknowledgement alone |

## Freshness contract

The existing expected-interval model remains authoritative.

| State | Rule | Example copy | Visual treatment |
|---|---|---|---|
| Fresh | `<= 2 × expected interval` | `2 秒前更新` | Solid symbol and normal emphasis |
| Delayed | `<= 5 × expected interval` | `数据延迟 · 14 秒前更新` | Clock icon, labelled amber accent |
| Stale | `> 5 × expected interval` | `数据可能已过期 · 48 秒未更新` | Clock-off icon, muted values, contextual caution |
| Offline | Broker/business timeout evidence | `设备离线 · 最后上报 4 分钟前` | Broken-link icon, neutral/danger severity only if a separate alert exists |
| Unavailable | No usable timestamp | `暂无遥测数据` | Dashed placeholder and explanation |

`Offline` is not automatically a `Critical` health condition. A critical
condition needs the named alarm or rule that made it critical.

## Severity contract

| Severity | Meaning | Required encodings | Default operator path |
|---|---|---|---|
| Info | Recorded operational fact | Info icon, label, timestamp | Inspect or dismiss from attention |
| Warning | Attention is needed | Warning icon, label, timestamp, short reason | Open incident context |
| Critical | Immediate handling is required | Critical icon, label, timestamp, reason, map/list context | Acknowledge ownership, inspect, follow SOP |

Colour supports but cannot carry this meaning alone. This is required by the
project accessibility baseline and [WCAG 2.2 SC 1.4.1](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color).

## Interaction rules

- A new critical alarm pulses its map ring once, increments the persistent
  alarm count, inserts in the queue, and announces a concise accessible status.
- An alert never focuses itself over a text field, modal, or command review.
- Selecting an incident opens its context; acknowledging it records ownership
  and does not resolve it.
- Telemetry values freeze in semantic meaning when stale: they remain visible
  as last known evidence but are never labelled real-time.
- Only capability- and authority-allowed actions render as enabled controls.
  Otherwise show the relevant reason and approved next path.

## Copy anatomy

Every non-normal state follows this order:

```text
Outcome · evidence time
Why it matters or the named condition
Safe next step
```

Example:

```text
数据可能已过期 · 48 秒未更新
AIRCRAFT-001 的位置和电量可能不是当前值。
查看连接状态
```
