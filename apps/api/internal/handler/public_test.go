package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newPublicHandler(svc *mockPublicSvc) *handler.PublicHandler {
	return handler.NewPublicHandler(svc)
}

func TestPublicHandler_GetProfile(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		mockFn     func(context.Context, string) (*domain.PublicProfile, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			slug: "mi-salon",
			mockFn: func(_ context.Context, slug string) (*domain.PublicProfile, error) {
				return &domain.PublicProfile{
					Slug: slug,
					Name: "Mi Salón de Belleza",
					Services: []*domain.Service{
						{Name: "Corte de cabello", DurationMin: 30},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "Mi Salón de Belleza",
		},
		{
			name: "not found",
			slug: "unknown-salon",
			mockFn: func(_ context.Context, _ string) (*domain.PublicProfile, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "encontrado",
		},
		{
			name: "service error",
			slug: "any-salon",
			mockFn: func(_ context.Context, _ string) (*domain.PublicProfile, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newPublicHandler(&mockPublicSvc{getProfileFn: tc.mockFn})
			app := newPublicApp()
			app.Get("/public/:slug", h.GetProfile)

			resp := doRequest(app, "GET", "/public/"+tc.slug)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestPublicHandler_GetAvailability(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		query      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			slug:       "mi-salon",
			query:      "?professional_id=" + validProfID + "&service_id=" + validServiceID + "&date=2026-03-20",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "missing professional_id",
			slug:       "mi-salon",
			query:      "?service_id=" + validServiceID + "&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "professional_id",
		},
		{
			name:       "invalid professional_id uuid",
			slug:       "mi-salon",
			query:      "?professional_id=bad-uuid&service_id=" + validServiceID + "&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "professional_id",
		},
		{
			name:       "missing service_id",
			slug:       "mi-salon",
			query:      "?professional_id=" + validProfID + "&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "service_id",
		},
		{
			name:       "invalid service_id uuid",
			slug:       "mi-salon",
			query:      "?professional_id=" + validProfID + "&service_id=bad&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "service_id",
		},
		{
			name:       "missing date",
			slug:       "mi-salon",
			query:      "?professional_id=" + validProfID + "&service_id=" + validServiceID,
			wantStatus: http.StatusBadRequest,
			wantBody:   "date",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newPublicHandler(&mockPublicSvc{})
			app := newPublicApp()
			app.Get("/public/:slug/availability", h.GetAvailability)

			resp := doRequest(app, "GET", "/public/"+tc.slug+"/availability"+tc.query)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestPublicHandler_Book(t *testing.T) {
	validBookBody := `{
		"professional_id":"` + validProfID + `",
		"service_id":"` + validServiceID + `",
		"starts_at":"2026-03-20T10:00:00Z",
		"customer_name":"María García",
		"customer_phone":"+18091234567"
	}`

	tests := []struct {
		name       string
		slug       string
		body       string
		mockFn     func(context.Context, string, *domain.CreateAppointmentRequest) (*domain.Appointment, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			slug: "mi-salon",
			body: validBookBody,
			mockFn: func(_ context.Context, _ string, _ *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
				return &domain.Appointment{ID: uuid.New()}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			slug:       "mi-salon",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name: "missing professional_id",
			slug: "mi-salon",
			body: `{
				"service_id":"` + validServiceID + `",
				"starts_at":"2026-03-20T10:00:00Z",
				"customer_name":"María"
			}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "professional_id",
		},
		{
			name: "missing service_id",
			slug: "mi-salon",
			body: `{
				"professional_id":"` + validProfID + `",
				"starts_at":"2026-03-20T10:00:00Z",
				"customer_name":"María"
			}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "service_id",
		},
		{
			name: "missing starts_at",
			slug: "mi-salon",
			body: `{
				"professional_id":"` + validProfID + `",
				"service_id":"` + validServiceID + `",
				"customer_name":"María",
				"customer_phone":"+18091234567"
			}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "starts_at",
		},
		{
			name: "missing customer info",
			slug: "mi-salon",
			body: `{
				"professional_id":"` + validProfID + `",
				"service_id":"` + validServiceID + `",
				"starts_at":"2026-03-20T10:00:00Z"
			}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "customer",
		},
		{
			name: "slot unavailable",
			slug: "mi-salon",
			body: validBookBody,
			mockFn: func(_ context.Context, _ string, _ *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
				return nil, domain.ErrSlotUnavailable
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "business not found",
			slug: "unknown",
			body: validBookBody,
			mockFn: func(_ context.Context, _ string, _ *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newPublicHandler(&mockPublicSvc{bookFn: tc.mockFn})
			app := newPublicApp()
			app.Post("/public/:slug/book", h.Book)

			resp := doJSON(app, "POST", "/public/"+tc.slug+"/book", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
