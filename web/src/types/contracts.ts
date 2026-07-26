export type DeviceStatus = 'REGISTERED' | 'ONLINE' | 'OFFLINE'
export type AlarmSeverity = 'INFO' | 'WARNING' | 'CRITICAL'
export type AlarmStatus = 'OPEN' | 'ACKNOWLEDGED' | 'RESOLVED'
export type CommandStatus =
  | 'CREATED'
  | 'VALIDATED'
  | 'REJECTED'
  | 'PUBLISH_PENDING'
  | 'PUBLISHED'
  | 'ACCEPTED'
  | 'EXECUTING'
  | 'SUCCEEDED'
  | 'FAILED'
  | 'TIMEOUT'
  | 'CANCELED'

export interface Device {
  id: string
  workspace_id: string
  vendor: string
  serial_number: string
  product_model?: string
  device_type: 'GATEWAY' | 'AIRCRAFT' | 'UNKNOWN'
  status: DeviceStatus
  updated_at?: string
  state_version?: number
  online?: boolean
  latitude?: number | null
  longitude?: number | null
  altitude?: number | null
  battery_percent?: number | null
  mode?: string
  server_time?: string
  payload?: Record<string, unknown>
}

export interface Alarm {
  id: string
  workspace_id: string
  device_id: string
  dedup_key: string
  alarm_type: string
  severity: AlarmSeverity
  status: AlarmStatus
  first_occurred_at: string
  last_occurred_at: string
  occurrence_count: number
  acknowledged_by?: string
  acknowledged_at?: string
  resolved_at?: string
  details?: Record<string, unknown>
}

export interface Command {
  id: string
  workspace_id: string
  target_device_id: string
  gateway_device_id?: string
  method: string
  status: CommandStatus
  risk_level: 'LOW' | 'MEDIUM' | 'HIGH'
  idempotency_key: string
  dji_tid?: string
  dji_bid?: string
  parameters?: Record<string, unknown>
  requested_by?: string
  result_code?: number
  result_message?: string
  created_at: string
  expires_at?: string
  completed_at?: string
  updated_at?: string
}

export interface DomainEvent {
  event_id: string
  event_type: string
  schema_version?: string
  workspace_id: string
  device_id?: string
  occurred_at: string
  received_at?: string
  sequence?: number
  payload?: Record<string, unknown>
  correlation?: { tid?: string; bid?: string; command_id?: string }
}

export interface WebSocketEnvelope {
  event_id: string
  type: string
  schema_version: string
  workspace_id: string
  aggregate_id?: string
  occurred_at: string
  sequence?: number
  request_id?: string
  data: Record<string, unknown>
}
