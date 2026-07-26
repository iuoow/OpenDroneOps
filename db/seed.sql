INSERT INTO workspaces (id, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'Local Demo')
ON CONFLICT DO NOTHING;

INSERT INTO devices (
  id, workspace_id, vendor, serial_number, gateway_serial_number,
  product_model, device_type, status, capabilities
)
VALUES
('00000000-0000-0000-0000-000000000101',
 '00000000-0000-0000-0000-000000000001',
 'DJI','SIM-DOCK-001','SIM-DOCK-001','SIMULATED_DOCK','GATEWAY','REGISTERED',
 '{"telemetry":true,"commands":["sim_status_refresh"]}'::jsonb),
('00000000-0000-0000-0000-000000000102',
 '00000000-0000-0000-0000-000000000001',
 'DJI','SIM-AIRCRAFT-001','SIM-DOCK-001','SIMULATED_AIRCRAFT','AIRCRAFT','REGISTERED',
 '{"telemetry":true,"trajectory":true}'::jsonb)
ON CONFLICT DO NOTHING;
