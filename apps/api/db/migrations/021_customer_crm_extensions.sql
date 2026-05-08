-- 021_customer_crm_extensions.sql
-- Campos CRM adicionales en la tabla customers

ALTER TABLE customers ADD COLUMN stage_id UUID REFERENCES pipeline_stages(id);
ALTER TABLE customers ADD COLUMN last_visit_at TIMESTAMPTZ;
ALTER TABLE customers ADD COLUMN next_recall_at TIMESTAMPTZ;
ALTER TABLE customers ADD COLUMN lifetime_value NUMERIC(10,2) DEFAULT 0;
ALTER TABLE customers ADD COLUMN acquisition_source TEXT;

CREATE INDEX idx_customers_stage       ON customers(tenant_id, stage_id);
CREATE INDEX idx_customers_recall      ON customers(tenant_id, next_recall_at) WHERE next_recall_at IS NOT NULL;
CREATE INDEX idx_customers_ltv         ON customers(tenant_id, lifetime_value DESC);
