-- Tabla de eventos analíticos particionada por mes.
-- Best-effort: si falla la inserción, no rompe la operación de negocio.

CREATE TABLE events (
    id          UUID DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    event_type  TEXT NOT NULL,
    actor_type  TEXT NOT NULL,
    actor_id    UUID,
    entity_type TEXT NOT NULL,
    entity_id   UUID NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, occurred_at)
) PARTITION BY RANGE (occurred_at);

-- Particiones: mes actual + 2 meses siguientes
CREATE TABLE events_y2026m05 PARTITION OF events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE events_y2026m06 PARTITION OF events
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE events_y2026m07 PARTITION OF events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

-- Indices (propagados automáticamente a cada partición)
CREATE INDEX idx_events_tenant_time
    ON events (tenant_id, occurred_at DESC);

CREATE INDEX idx_events_tenant_type_time
    ON events (tenant_id, event_type, occurred_at DESC);

CREATE INDEX idx_events_tenant_entity
    ON events (tenant_id, entity_type, entity_id);

CREATE INDEX idx_events_payload
    ON events USING GIN (payload);

-- RLS
ALTER TABLE events ENABLE ROW LEVEL SECURITY;

CREATE POLICY events_tenant_isolation ON events
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
