// Package service contiene la lógica de negocio — orquesta repositorios y servicios externos.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/config"
	"github.com/citaspot/api/internal/domain"
)

// authService implementa domain.AuthService.
type authService struct {
	repo   domain.AuthRepository
	cfg    *config.Config
	client *http.Client
}

// NewAuthService crea el servicio de autenticación.
func NewAuthService(repo domain.AuthRepository, cfg *config.Config) domain.AuthService {
	return &authService{
		repo:   repo,
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// ── Supabase Admin API helpers ────────────────────────────────────────────────

// supabaseAdminUser representa la respuesta de la Admin API al crear un usuario.
type supabaseAdminUser struct {
	ID string `json:"id"`
}

// supabaseTokenResponse representa la respuesta de /auth/v1/token.
type supabaseTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// supabaseErrorResponse representa un error de la Supabase Auth API.
type supabaseErrorResponse struct {
	Msg  string `json:"msg"`
	Code string `json:"error_code"`
}

// createSupabaseUser llama a la Admin API para crear un usuario en Supabase Auth.
// Requiere SERVICE_ROLE_KEY — nunca debe llamarse desde el cliente.
func (s *authService) createSupabaseUser(ctx context.Context, email, password string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"email":          email,
		"password":       password,
		"email_confirm":  true, // confirmación automática en registro de tenant
	})

	url := s.cfg.SupabaseURL + "/auth/v1/admin/users"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("authService.createSupabaseUser: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.cfg.SupabaseServiceKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseServiceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("authService.createSupabaseUser: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnprocessableEntity {
		// El email ya existe en Supabase Auth
		return "", domain.ErrEmailAlreadyExists
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var supaErr supabaseErrorResponse
		_ = json.Unmarshal(respBody, &supaErr)
		return "", fmt.Errorf("authService.createSupabaseUser: supabase status %d: %s", resp.StatusCode, supaErr.Msg)
	}

	var user supabaseAdminUser
	if err := json.Unmarshal(respBody, &user); err != nil {
		return "", fmt.Errorf("authService.createSupabaseUser: decode: %w", err)
	}
	return user.ID, nil
}

// deleteSupabaseUser elimina un usuario de Supabase Auth en caso de rollback.
// Se llama cuando la inserción en la DB falla tras crear el usuario en Supabase.
func (s *authService) deleteSupabaseUser(ctx context.Context, authID string) {
	url := s.cfg.SupabaseURL + "/auth/v1/admin/users/" + authID
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("apikey", s.cfg.SupabaseServiceKey)
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupabaseServiceKey)
	resp, _ := s.client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
}

// signInWithPassword llama a Supabase para obtener tokens JWT.
func (s *authService) signInWithPassword(ctx context.Context, email, password string) (*supabaseTokenResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	url := s.cfg.SupabaseURL + "/auth/v1/token?grant_type=password"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("authService.signInWithPassword: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.cfg.SupabaseAnonKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authService.signInWithPassword: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusBadRequest {
		return nil, domain.ErrInvalidCredentials
	}
	if resp.StatusCode != http.StatusOK {
		var supaErr supabaseErrorResponse
		_ = json.Unmarshal(respBody, &supaErr)
		return nil, fmt.Errorf("authService.signInWithPassword: supabase status %d: %s", resp.StatusCode, supaErr.Msg)
	}

	var tokens supabaseTokenResponse
	if err := json.Unmarshal(respBody, &tokens); err != nil {
		return nil, fmt.Errorf("authService.signInWithPassword: decode: %w", err)
	}
	return &tokens, nil
}

// refreshSupabaseToken intercambia un refresh_token por tokens nuevos.
func (s *authService) refreshSupabaseToken(ctx context.Context, refreshToken string) (*supabaseTokenResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})

	url := s.cfg.SupabaseURL + "/auth/v1/token?grant_type=refresh_token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("authService.refreshSupabaseToken: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.cfg.SupabaseAnonKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authService.refreshSupabaseToken: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, domain.ErrInvalidToken
	}

	var tokens supabaseTokenResponse
	if err := json.Unmarshal(respBody, &tokens); err != nil {
		return nil, fmt.Errorf("authService.refreshSupabaseToken: decode: %w", err)
	}
	return &tokens, nil
}

// ── Slug helpers ──────────────────────────────────────────────────────────────

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// generateSlug convierte un nombre de negocio en un slug URL-safe.
// Ejemplo: "Salón Belleza & Spa" → "salon-belleza-spa"
func generateSlug(name string) string {
	s := strings.ToLower(name)
	// Reemplazar caracteres especiales latinos
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"ñ", "n", "ü", "u", "à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
	)
	s = replacer.Replace(s)
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// Máximo 60 caracteres
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	return s
}

// uniqueSlug genera un slug único, añadiendo sufijo numérico si ya existe.
func (s *authService) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; i <= 99; i++ {
		exists, err := s.repo.TenantSlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	// Fallback con UUID suffix si hay muchos conflictos
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:8]), nil
}

// ── domain.AuthService implementation ────────────────────────────────────────

// Register crea un nuevo tenant + usuario propietario.
// Flujo: slug → Supabase Admin API → DB transaction (tenant + user) → sign-in → tokens
func (s *authService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.RegisterResponse, error) {
	// 1. Generar slug único para el negocio
	baseSlug := generateSlug(req.BusinessName)
	slug, err := s.uniqueSlug(ctx, baseSlug)
	if err != nil {
		return nil, fmt.Errorf("authService.Register: slug: %w", err)
	}

	// 2. Crear usuario en Supabase Auth (antes de la DB para obtener auth_id)
	authID, err := s.createSupabaseUser(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	// 3. Insertar tenant y usuario en la DB
	// Trial de 30 días sin tarjeta de crédito requerida
	trialEndsAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	tenant := &domain.Tenant{
		ID:           uuid.New(),
		Slug:         slug,
		Name:         req.BusinessName,
		BusinessType: req.BusinessType,
		Email:        req.Email,
		City:         req.City,
		Country:      req.Country,
		Timezone:     req.Timezone,
		Plan:         "trial",
		PlanStatus:   "trial",
		TrialEndsAt:  &trialEndsAt,
	}

	if err := s.repo.CreateTenant(ctx, tenant); err != nil {
		// Rollback: eliminar usuario de Supabase Auth
		s.deleteSupabaseUser(ctx, authID)
		return nil, fmt.Errorf("authService.Register: create tenant: %w", err)
	}

	user := &domain.User{
		ID:       uuid.New(),
		TenantID: tenant.ID,
		Email:    req.Email,
		Name:     req.Name,
		Role:     "owner",
		AuthID:   authID,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		// Rollback: eliminar usuario de Supabase Auth
		// (el tenant queda huérfano temporalmente — una tarea de limpieza puede recolectarlo)
		s.deleteSupabaseUser(ctx, authID)
		return nil, fmt.Errorf("authService.Register: create user: %w", err)
	}

	// 4. Sign-in para obtener tokens JWT
	tokens, err := s.signInWithPassword(ctx, req.Email, req.Password)
	if err != nil {
		// El usuario ya está creado — no hacemos rollback aquí, puede hacer login después
		return nil, fmt.Errorf("authService.Register: sign-in: %w", err)
	}

	return &domain.RegisterResponse{
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: domain.UserDTO{
			ID:       user.ID,
			TenantID: tenant.ID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
		},
		Tenant: domain.TenantDTO{
			ID:           tenant.ID,
			Slug:         tenant.Slug,
			Name:         tenant.Name,
			BusinessType: tenant.BusinessType,
			Plan:         tenant.Plan,
			PlanStatus:   tenant.PlanStatus,
			TrialEndsAt:  tenant.TrialEndsAt,
		},
	}, nil
}

// Login autentica un usuario existente y retorna tokens JWT.
func (s *authService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	// 1. Sign-in con Supabase para validar credenciales
	tokens, err := s.signInWithPassword(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	// 2. Resolver usuario y tenant desde nuestra DB
	user, tenant, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: domain.UserDTO{
			ID:       user.ID,
			TenantID: user.TenantID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     user.Role,
		},
		Tenant: domain.TenantDTO{
			ID:           tenant.ID,
			Slug:         tenant.Slug,
			Name:         tenant.Name,
			BusinessType: tenant.BusinessType,
			Plan:         tenant.Plan,
			PlanStatus:   tenant.PlanStatus,
			TrialEndsAt:  tenant.TrialEndsAt,
		},
	}, nil
}

// RefreshToken renueva el access_token usando el refresh_token de Supabase.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*domain.RefreshResponse, error) {
	tokens, err := s.refreshSupabaseToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return &domain.RefreshResponse{
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
