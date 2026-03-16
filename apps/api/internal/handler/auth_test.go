package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
)

func TestAuthHandler_Register(t *testing.T) {
	validBody := `{"email":"owner@test.com","password":"password123","name":"John Doe","business_name":"Salon Pro","business_type":"beauty"}`

	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, *domain.RegisterRequest) (*domain.RegisterResponse, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: validBody,
			mockFn: func(_ context.Context, req *domain.RegisterRequest) (*domain.RegisterResponse, error) {
				return &domain.RegisterResponse{Token: "access-token"}, nil
			},
			wantStatus: http.StatusCreated,
			wantBody:   "access-token",
		},
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing required fields",
			body:       `{"email":"test@test.com"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "password too short",
			body:       `{"email":"t@t.com","password":"short","name":"John","business_name":"Salon","business_type":"beauty"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "invalid business_type",
			body:       `{"email":"t@t.com","password":"password123","name":"John","business_name":"Salon","business_type":"restaurant"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "email already exists",
			body: validBody,
			mockFn: func(_ context.Context, _ *domain.RegisterRequest) (*domain.RegisterResponse, error) {
				return nil, domain.ErrEmailAlreadyExists
			},
			wantStatus: http.StatusConflict,
			wantBody:   "email",
		},
		{
			name: "slug already in use",
			body: validBody,
			mockFn: func(_ context.Context, _ *domain.RegisterRequest) (*domain.RegisterResponse, error) {
				return nil, domain.ErrSlugAlreadyExists
			},
			wantStatus: http.StatusConflict,
			wantBody:   "nombre",
		},
		{
			name:       "invalid email format",
			body:       `{"email":"not-an-email","password":"password123","name":"John","business_name":"Salon","business_type":"beauty"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "email",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewAuthHandler(&mockAuthSvc{registerFn: tc.mockFn})
			app := newPublicApp()
			app.Post("/auth/register", h.Register)

			resp := doJSON(app, "POST", "/auth/register", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, *domain.LoginRequest) (*domain.LoginResponse, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: `{"email":"owner@test.com","password":"password123"}`,
			mockFn: func(_ context.Context, _ *domain.LoginRequest) (*domain.LoginResponse, error) {
				return &domain.LoginResponse{Token: "access-token"}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "access-token",
		},
		{
			name:       "invalid json",
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing email",
			body:       `{"password":"password123"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "invalid email format",
			body:       `{"email":"not-email","password":"password123"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "email",
		},
		{
			name: "wrong credentials",
			body: `{"email":"owner@test.com","password":"wrongpass"}`,
			mockFn: func(_ context.Context, _ *domain.LoginRequest) (*domain.LoginResponse, error) {
				return nil, domain.ErrInvalidCredentials
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "contraseña",
		},
		{
			name: "user not found",
			body: `{"email":"nobody@test.com","password":"password123"}`,
			mockFn: func(_ context.Context, _ *domain.LoginRequest) (*domain.LoginResponse, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "encontrado",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewAuthHandler(&mockAuthSvc{loginFn: tc.mockFn})
			app := newPublicApp()
			app.Post("/auth/login", h.Login)

			resp := doJSON(app, "POST", "/auth/login", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, string) (*domain.RefreshResponse, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: `{"refresh_token":"valid-refresh-token"}`,
			mockFn: func(_ context.Context, _ string) (*domain.RefreshResponse, error) {
				return &domain.RefreshResponse{Token: "new-access-token"}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "new-access-token",
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing refresh_token",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "refresh_token",
		},
		{
			name: "expired token",
			body: `{"refresh_token":"expired-token"}`,
			mockFn: func(_ context.Context, _ string) (*domain.RefreshResponse, error) {
				return nil, domain.ErrInvalidToken
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "inválido",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewAuthHandler(&mockAuthSvc{refreshTokenFn: tc.mockFn})
			app := newPublicApp()
			app.Post("/auth/refresh", h.RefreshToken)

			resp := doJSON(app, "POST", "/auth/refresh", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
