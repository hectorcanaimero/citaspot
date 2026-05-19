package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

// waitlistRepository implementa domain.WaitlistRepository.
// La tabla `waitlist_signups` es GLOBAL (sin tenant_id, sin RLS) — no usa
// withTenant() como las demás tablas.
type waitlistRepository struct {
	db *pgxpool.Pool
}

// NewWaitlistRepository crea el repositorio de la lista de espera pre-launch.
func NewWaitlistRepository(db *pgxpool.Pool) domain.WaitlistRepository {
	return &waitlistRepository{db: db}
}

// Create inserta un signup. Devuelve ErrWaitlistEmailExists si el email ya existe.
// Acepta ip_address y user_agent opcionales (vacío → NULL).
func (r *waitlistRepository) Create(ctx context.Context, signup *domain.WaitlistSignup) error {
	// nullable helpers: pasar nil al driver cuando el string está vacío
	var ipPtr *string
	if signup.IPAddress != "" {
		ip := signup.IPAddress
		ipPtr = &ip
	}
	var uaPtr *string
	if signup.UserAgent != "" {
		ua := signup.UserAgent
		uaPtr = &ua
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO waitlist_signups (id, email, business_name, ip_address, user_agent)
		VALUES ($1, $2, $3, $4::inet, $5)
		RETURNING created_at
	`, signup.ID, signup.Email, signup.BusinessName, ipPtr, uaPtr).Scan(&signup.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Violación de UNIQUE en email (waitlist_signups_email_key)
			return domain.ErrWaitlistEmailExists
		}
		return fmt.Errorf("waitlistRepository.Create: %w", err)
	}
	return nil
}
