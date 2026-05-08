package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultHealthcarePipelineStages — para verticales clínicos (wellness, clinic).
var DefaultHealthcarePipelineStages = []struct {
	Name     string
	Position int
	Color    string
}{
	{Name: "Nuevo paciente", Position: 0, Color: "#6366f1"},
	{Name: "Primera consulta", Position: 1, Color: "#8b5cf6"},
	{Name: "En tratamiento", Position: 2, Color: "#10b981"},
	{Name: "Alta médica", Position: 3, Color: "#22c55e"},
}

// DefaultServicePipelineStages — para verticales de servicios (beauty, barbershop).
var DefaultServicePipelineStages = []struct {
	Name     string
	Position int
	Color    string
}{
	{Name: "Lead", Position: 0, Color: "#6366f1"},
	{Name: "Interesado", Position: 1, Color: "#8b5cf6"},
	{Name: "Cliente activo", Position: 2, Color: "#10b981"},
	{Name: "Cliente recurrente", Position: 3, Color: "#22c55e"},
}

// pipelineStagesForBusinessType devuelve el template de etapas para el vertical dado.
// Si el business_type no tiene template propio, retorna nil.
func pipelineStagesForBusinessType(businessType string) []struct {
	Name     string
	Position int
	Color    string
} {
	switch businessType {
	case "dental":
		return DefaultDentalPipelineStages
	case "wellness", "clinic":
		return DefaultHealthcarePipelineStages
	case "beauty", "barbershop":
		return DefaultServicePipelineStages
	default:
		return nil
	}
}

// SeedPipelineForBusinessType seedea el template de etapas para el tenant
// según su vertical, solo si NO tiene stages aún. Idempotente.
//
// Retorna el slice de etapas seedeadas. Si el tenant ya tenía stages o el
// vertical no tiene template, retorna nil.
func SeedPipelineForBusinessType(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
	businessType string,
) ([]string, error) {
	template := pipelineStagesForBusinessType(businessType)
	if template == nil {
		return nil, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("seed.SeedPipelineForBusinessType: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return nil, fmt.Errorf("seed.SeedPipelineForBusinessType: set_config: %w", err)
	}

	// Si ya hay etapas, no pisamos nada
	var existing int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM pipeline_stages WHERE tenant_id = $1", tenantID).Scan(&existing); err != nil {
		return nil, fmt.Errorf("seed.SeedPipelineForBusinessType: count: %w", err)
	}
	if existing > 0 {
		return nil, nil
	}

	seeded := make([]string, 0, len(template))
	for i, s := range template {
		isDefault := i == 0
		_, err := tx.Exec(ctx, `
			INSERT INTO pipeline_stages (id, tenant_id, name, position, color, is_default, auto_rules_enabled, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW())
			ON CONFLICT (tenant_id, position) DO NOTHING
		`, uuid.New(), tenantID, s.Name, s.Position, s.Color, isDefault)
		if err != nil {
			return nil, fmt.Errorf("seed.SeedPipelineForBusinessType: stage '%s': %w", s.Name, err)
		}
		seeded = append(seeded, s.Name)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("seed.SeedPipelineForBusinessType: commit: %w", err)
	}
	return seeded, nil
}
