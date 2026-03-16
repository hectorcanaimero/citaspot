-- 011_notification_logs.sql
-- Registro de todos los mensajes de WhatsApp enviados por el sistema

CREATE TABLE notification_logs (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id),
  appointment_id  UUID REFERENCES appointments(id),
  customer_id     UUID REFERENCES customers(id),
  type            VARCHAR(50) NOT NULL,
                  -- 'confirmation','reminder_24h','reminder_2h','review_request','reengagement'
  wa_message_id   VARCHAR(255),
  status          VARCHAR(20) DEFAULT 'sent',  -- 'sent','delivered','failed'
  content         TEXT,
  sent_at         TIMESTAMPTZ DEFAULT NOW(),
  delivered_at    TIMESTAMPTZ,
  error_message   TEXT
);

CREATE INDEX idx_notif_tenant_type ON notification_logs(tenant_id, type, sent_at);
CREATE INDEX idx_notif_appointment ON notification_logs(appointment_id);
