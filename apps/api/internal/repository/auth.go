// Package repository contiene el acceso a la DB — solo queries SQL, sin lógica de negocio.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

// authRepository implementa domain.AuthRepository usando pgxpool.
type authRepository struct {
	db *pgxpool.Pool
}

// NewAuthRepository crea una instancia del repositorio de auth.
func NewAuthRepository(db *pgxpool.Pool) domain.AuthRepository {
	return &authRepository{db: db}
}

// CreateTenant inserta un nuevo tenant en la DB.
// tenant_id no usa RLS porque el tenant aún no existe — opera sin SET app.tenant_id.
func (r *authRepository) CreateTenant(ctx context.Context, t *domain.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, slug, name, business_type, email,
			city, country, timezone,
			plan, plan_status, trial_ends_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			NOW(), NOW()
		)
	`
	_, err := r.db.Exec(ctx, query,
		t.ID, t.Slug, t.Name, t.BusinessType, t.Email,
		t.City, t.Country, t.Timezone,
		t.Plan, t.PlanStatus, t.TrialEndsAt,
	)
	if err != nil {
		return fmt.Errorf("authRepository.CreateTenant: %w", err)
	}
	return nil
}

// CreateUser inserta un nuevo usuario en la DB.
// No requiere RLS — la operación de registro ocurre sin sesión de tenant todavía.
func (r *authRepository) CreateUser(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (id, tenant_id, email, name, role, auth_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.db.Exec(ctx, query,
		u.ID, u.TenantID, u.Email, u.Name, u.Role, u.AuthID,
	)
	if err != nil {
		return fmt.Errorf("authRepository.CreateUser: %w", err)
	}
	return nil
}

// FindUserByEmail retorna el usuario y su tenant por email.
// Usado en el flujo de login para validar que el usuario existe en la DB.
func (r *authRepository) FindUserByEmail(ctx context.Context, email string) (*domain.User, *domain.Tenant, error) {
	query := `
		SELECT
			u.id, u.tenant_id, u.email, u.name, u.role, u.auth_id, u.created_at,
			t.id, t.slug, t.name, t.business_type, t.email,
			t.city, t.country, t.timezone, t.plan, t.plan_status, t.onboarding_done, t.trial_ends_at,
			t.wa_status, COALESCE(t.settings, '{}')::TEXT,
			t.created_at, t.updated_at
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.email = $1
		LIMIT 1
	`
	return r.scanUserWithTenant(ctx, query, email)
}

// FindUserByAuthID retorna el usuario y su tenant por el Supabase Auth UID.
// Usado en el middleware JWT para resolver el tenant de cada request.
func (r *authRepository) FindUserByAuthID(ctx context.Context, authID string) (*domain.User, *domain.Tenant, error) {
	query := `
		SELECT
			u.id, u.tenant_id, u.email, u.name, u.role, u.auth_id, u.created_at,
			t.id, t.slug, t.name, t.business_type, t.email,
			t.city, t.country, t.timezone, t.plan, t.plan_status, t.onboarding_done, t.trial_ends_at,
			t.wa_status, COALESCE(t.settings, '{}')::TEXT,
			t.created_at, t.updated_at
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.auth_id = $1
		LIMIT 1
	`
	return r.scanUserWithTenant(ctx, query, authID)
}

// TenantSlugExists retorna true si el slug ya existe en la DB.
func (r *authRepository) TenantSlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)", slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("authRepository.TenantSlugExists: %w", err)
	}
	return exists, nil
}

// FindTenantByID retorna un tenant por su ID.
func (r *authRepository) FindTenantByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, slug, name, business_type, email,
		       city, country, timezone, plan, plan_status, onboarding_done, trial_ends_at,
		       wa_status, COALESCE(settings, '{}')::TEXT,
		       created_at, updated_at
		FROM tenants WHERE id = $1
	`
	t := &domain.Tenant{}
	var waStatus *string
	var settingsRaw []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Slug, &t.Name, &t.BusinessType, &t.Email,
		&t.City, &t.Country, &t.Timezone, &t.Plan, &t.PlanStatus, &t.OnboardingDone, &t.TrialEndsAt,
		&waStatus, &settingsRaw,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("authRepository.FindTenantByID: %w", err)
	}
	if waStatus != nil {
		t.WAStatus = *waStatus
	}
	_ = json.Unmarshal(settingsRaw, &t.Settings)
	return t, nil
}

// FindTenantBySlug retorna un tenant por su slug público.
func (r *authRepository) FindTenantBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `
		SELECT id, slug, name, business_type, email,
		       city, country, timezone, plan, plan_status, onboarding_done, trial_ends_at,
		       wa_status, COALESCE(settings, '{}')::TEXT,
		       created_at, updated_at
		FROM tenants WHERE slug = $1
	`
	t := &domain.Tenant{}
	var waStatus *string
	var settingsRaw []byte
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&t.ID, &t.Slug, &t.Name, &t.BusinessType, &t.Email,
		&t.City, &t.Country, &t.Timezone, &t.Plan, &t.PlanStatus, &t.OnboardingDone, &t.TrialEndsAt,
		&waStatus, &settingsRaw,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("authRepository.FindTenantBySlug: %w", err)
	}
	if waStatus != nil {
		t.WAStatus = *waStatus
	}
	_ = json.Unmarshal(settingsRaw, &t.Settings)
	return t, nil
}

// UpdateTenantBilling actualiza plan, plan_status y los IDs de Stripe del tenant.
// Se llama desde el webhook handler tras recibir eventos de Stripe.
func (r *authRepository) UpdateTenantBilling(ctx context.Context, tenantID uuid.UUID, plan, planStatus, stripeCustomerID, stripeSubID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tenants
		    SET plan = $2, plan_status = $3,
		        stripe_customer_id = NULLIF($4, ''),
		        stripe_sub_id      = NULLIF($5, ''),
		        updated_at = NOW()
		  WHERE id = $1`,
		tenantID, plan, planStatus, stripeCustomerID, stripeSubID,
	)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateTenantBilling: %w", err)
	}
	return nil
}

// FindTenantStripeIDs retorna stripe_customer_id y stripe_sub_id del tenant.
func (r *authRepository) FindTenantStripeIDs(ctx context.Context, tenantID uuid.UUID) (customerID, subID string, err error) {
	var cID, sID *string
	err = r.db.QueryRow(ctx,
		`SELECT stripe_customer_id, stripe_sub_id FROM tenants WHERE id = $1`, tenantID,
	).Scan(&cID, &sID)
	if err != nil {
		return "", "", fmt.Errorf("authRepository.FindTenantStripeIDs: %w", err)
	}
	if cID != nil {
		customerID = *cID
	}
	if sID != nil {
		subID = *sID
	}
	return customerID, subID, nil
}

// FindConnectedTenantSlugs retorna los slugs de todos los tenants con wa_status = 'connected'.
// Se usa al startup para re-registrar webhooks en Evolution API sin depender de que ella los persista.
func (r *authRepository) FindConnectedTenantSlugs(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, "SELECT slug FROM tenants WHERE wa_status = 'connected'")
	if err != nil {
		return nil, fmt.Errorf("authRepository.FindConnectedTenantSlugs: %w", err)
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("authRepository.FindConnectedTenantSlugs: scan: %w", err)
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

// CompleteOnboarding marca el onboarding del tenant como completado.
func (r *authRepository) CompleteOnboarding(ctx context.Context, tenantID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE tenants SET onboarding_done = TRUE, updated_at = NOW() WHERE id = $1",
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("authRepository.CompleteOnboarding: %w", err)
	}
	return nil
}

// UpdateTenantWAStatus actualiza el campo wa_status del tenant identificado por slug.
// No requiere RLS — la tabla tenants es global y no tiene Row Level Security.
func (r *authRepository) UpdateTenantWAStatus(ctx context.Context, slug, status string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE tenants SET wa_status = $2, updated_at = NOW() WHERE slug = $1",
		slug, status,
	)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateTenantWAStatus: %w", err)
	}
	return nil
}

// scanUserWithTenant escanea una fila que incluye user + tenant en una sola query.
func (r *authRepository) scanUserWithTenant(ctx context.Context, query, arg string) (*domain.User, *domain.Tenant, error) {
	u := &domain.User{}
	t := &domain.Tenant{}
	var trialEndsAt *time.Time
	var waStatus *string
	var settingsRaw []byte

	err := r.db.QueryRow(ctx, query, arg).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &u.AuthID, &u.CreatedAt,
		&t.ID, &t.Slug, &t.Name, &t.BusinessType, &t.Email,
		&t.City, &t.Country, &t.Timezone, &t.Plan, &t.PlanStatus, &t.OnboardingDone, &trialEndsAt,
		&waStatus, &settingsRaw,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domain.ErrNotFound
		}
		return nil, nil, fmt.Errorf("authRepository.scanUserWithTenant: %w", err)
	}
	t.TrialEndsAt = trialEndsAt
	if waStatus != nil {
		t.WAStatus = *waStatus
	}
	_ = json.Unmarshal(settingsRaw, &t.Settings)
	return u, t, nil
}

// GetTenantSettings retorna los settings JSONB del tenant.
func (r *authRepository) GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSettings, error) {
	var raw []byte
	err := r.db.QueryRow(ctx,
		"SELECT COALESCE(settings, '{}')::TEXT FROM tenants WHERE id = $1", tenantID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("authRepository.GetTenantSettings: %w", err)
	}
	var s domain.TenantSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("authRepository.GetTenantSettings: unmarshal: %w", err)
	}
	return &s, nil
}

// UpdateTenantSettings actualiza los settings JSONB del tenant.
func (r *authRepository) UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, s *domain.TenantSettings) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateTenantSettings: marshal: %w", err)
	}
	tag, err := r.db.Exec(ctx,
		"UPDATE tenants SET settings = $2, updated_at = NOW() WHERE id = $1",
		tenantID, raw,
	)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateTenantSettings: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
