package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// emailRegex valida formato básico de email (RFC 5322 simplificado).
// Suficiente para capturar leads — la validación profunda la haría un
// servicio de verificación dedicado, fuera del alcance del waitlist.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type waitlistSvc struct {
	repo domain.WaitlistRepository
}

// NewWaitlistSvc crea el servicio de lista de espera pre-launch.
func NewWaitlistSvc(repo domain.WaitlistRepository) domain.WaitlistSvc {
	return &waitlistSvc{repo: repo}
}

// Join normaliza, valida y persiste un lead en la lista de espera.
// Reglas:
//   - email: trim + lowercase + regex
//   - business_name: trim + min 2 chars + max 120
//
// No loggea el email completo (PII) — solo el dominio para diagnóstico.
func (s *waitlistSvc) Join(ctx context.Context, input *domain.JoinWaitlistInput) (*domain.WaitlistSignup, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	businessName := strings.TrimSpace(input.BusinessName)

	if email == "" || !emailRegex.MatchString(email) {
		return nil, domain.ErrWaitlistInvalidEmail
	}
	if len(email) > 254 {
		return nil, domain.ErrWaitlistInvalidEmail
	}
	if len(businessName) < 2 {
		return nil, fmt.Errorf("nombre del negocio requerido: %w", domain.ErrValidation)
	}
	if len(businessName) > 120 {
		return nil, fmt.Errorf("nombre del negocio demasiado largo: %w", domain.ErrValidation)
	}

	signup := &domain.WaitlistSignup{
		ID:           uuid.New(),
		Email:        email,
		BusinessName: businessName,
		IPAddress:    strings.TrimSpace(input.IPAddress),
		UserAgent:    strings.TrimSpace(input.UserAgent),
	}

	if err := s.repo.Create(ctx, signup); err != nil {
		return nil, err
	}

	// Log sin PII: solo dominio del email + ID generado para correlación
	slog.Info("waitlist.signup created",
		"signup_id", signup.ID.String(),
		"email_domain", emailDomain(email),
	)

	return signup, nil
}

// emailDomain extrae el dominio del email para loggear sin exponer el local-part.
// Asume email ya validado por el regex.
func emailDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return "unknown"
	}
	return email[at+1:]
}
