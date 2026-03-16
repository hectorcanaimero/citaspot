package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newSvcHandler(svc *mockServiceSvc) *handler.ServiceHandler {
	return handler.NewServiceHandler(svc)
}

func TestServiceHandler_List(t *testing.T) {
	t.Run("success empty list", func(t *testing.T) {
		h := newSvcHandler(&mockServiceSvc{})
		app := newProtectedApp()
		app.Get("/services", h.List)

		resp := doRequest(app, "GET", "/services")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "data")
	})

	t.Run("success with services", func(t *testing.T) {
		h := newSvcHandler(&mockServiceSvc{
			listFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Service, error) {
				return []*domain.Service{
					{ID: testID, Name: "Corte de cabello", DurationMin: 30},
				}, nil
			},
		})
		app := newProtectedApp()
		app.Get("/services", h.List)

		resp := doRequest(app, "GET", "/services")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "Corte de cabello")
	})

	t.Run("service error returns 500", func(t *testing.T) {
		h := newSvcHandler(&mockServiceSvc{
			listFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Service, error) {
				return nil, domain.ErrInternal
			},
		})
		app := newProtectedApp()
		app.Get("/services", h.List)

		resp := doRequest(app, "GET", "/services")
		assertStatus(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestServiceHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID, *domain.ServiceInput) (*domain.Service, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       `{"name":"Corte Premium","duration_min":45}`,
			wantStatus: http.StatusCreated,
			wantBody:   "Corte Premium",
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing name",
			body:       `{"duration_min":45}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "duration too short",
			body:       `{"name":"Corte","duration_min":3}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "duration too long",
			body:       `{"name":"Corte","duration_min":500}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "name too short",
			body:       `{"name":"A","duration_min":30}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newSvcHandler(&mockServiceSvc{createFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/services", h.Create)

			resp := doJSON(app, "POST", "/services", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestServiceHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Service, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   testID.String(),
		},
		{
			name:       "invalid uuid",
			id:         "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name: "not found",
			id:   testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Service, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newSvcHandler(&mockServiceSvc{getByIDFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/services/:id", h.GetByID)

			resp := doRequest(app, "GET", "/services/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestServiceHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.ServiceInput) (*domain.Service, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			body:       `{"name":"Corte Actualizado","duration_min":60}`,
			wantStatus: http.StatusOK,
			wantBody:   "Corte Actualizado",
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       `{"name":"Corte","duration_min":30}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name:       "invalid json",
			id:         testID.String(),
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name: "not found",
			id:   testID.String(),
			body: `{"name":"Corte","duration_min":30}`,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.ServiceInput) (*domain.Service, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newSvcHandler(&mockServiceSvc{updateFn: tc.mockFn})
			app := newProtectedApp()
			app.Patch("/services/:id", h.Update)

			resp := doJSON(app, "PATCH", "/services/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
