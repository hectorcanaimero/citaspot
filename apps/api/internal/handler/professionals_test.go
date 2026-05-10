package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newProfHandler(svc *mockProfessionalSvc) *handler.ProfessionalHandler {
	return handler.NewProfessionalHandler(svc)
}

func TestProfessionalHandler_List(t *testing.T) {
	t.Run("success empty list", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{})
		app := newProtectedApp()
		app.Get("/professionals", h.List)

		resp := doRequest(app, "GET", "/professionals")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "data")
	})

	t.Run("success with professionals", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{
			listFn: func(_ context.Context, _ uuid.UUID, _ bool) ([]*domain.Professional, error) {
				return []*domain.Professional{
					{ID: testID, Name: "Dr. García"},
				}, nil
			},
		})
		app := newProtectedApp()
		app.Get("/professionals", h.List)

		resp := doRequest(app, "GET", "/professionals")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "Dr. García")
	})

	t.Run("service error returns 500", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{
			listFn: func(_ context.Context, _ uuid.UUID, _ bool) ([]*domain.Professional, error) {
				return nil, domain.ErrInternal
			},
		})
		app := newProtectedApp()
		app.Get("/professionals", h.List)

		resp := doRequest(app, "GET", "/professionals")
		assertStatus(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestProfessionalHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID, *domain.ProfessionalInput) (*domain.Professional, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       `{"name":"Dr. Martínez","specialty":"Dermatología"}`,
			wantStatus: http.StatusCreated,
			wantBody:   "Dr. Martínez",
		},
		{
			name:       "invalid json",
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing name",
			body:       `{"specialty":"Dermatología"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "name too short",
			body:       `{"name":"A"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "conflict",
			body: `{"name":"Dr. Existente"}`,
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.ProfessionalInput) (*domain.Professional, error) {
				return nil, domain.ErrConflict
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfHandler(&mockProfessionalSvc{createFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/professionals", h.Create)

			resp := doJSON(app, "POST", "/professionals", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestProfessionalHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Professional, error)
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
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Professional, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "encontrado",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfHandler(&mockProfessionalSvc{getByIDFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/professionals/:id", h.GetByID)

			resp := doRequest(app, "GET", "/professionals/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestProfessionalHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.ProfessionalInput) (*domain.Professional, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			body:       `{"name":"Dr. Updated"}`,
			wantStatus: http.StatusOK,
			wantBody:   "Dr. Updated",
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       `{"name":"Dr. Updated"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name:       "invalid json",
			id:         testID.String(),
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name: "not found",
			id:   testID.String(),
			body: `{"name":"Dr. Updated"}`,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.ProfessionalInput) (*domain.Professional, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newProfHandler(&mockProfessionalSvc{updateFn: tc.mockFn})
			app := newProtectedApp()
			app.Patch("/professionals/:id", h.Update)

			resp := doJSON(app, "PATCH", "/professionals/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestProfessionalHandler_GetSchedule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{
			getScheduleFn: func(_ context.Context, _, _ uuid.UUID) ([]*domain.Schedule, error) {
				return []*domain.Schedule{
					{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00", IsActive: true},
				}, nil
			},
		})
		app := newProtectedApp()
		app.Get("/professionals/:id/schedule", h.GetSchedule)

		resp := doRequest(app, "GET", "/professionals/"+testID.String()+"/schedule")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "09:00")
	})

	t.Run("invalid uuid", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{})
		app := newProtectedApp()
		app.Get("/professionals/:id/schedule", h.GetSchedule)

		resp := doRequest(app, "GET", "/professionals/bad-uuid/schedule")
		assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestProfessionalHandler_SetSchedule(t *testing.T) {
	validBody := `[{"day_of_week":1,"start_time":"09:00","end_time":"17:00","is_active":true}]`

	t.Run("success", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{})
		app := newProtectedApp()
		app.Put("/professionals/:id/schedule", h.SetSchedule)

		resp := doJSON(app, "PUT", "/professionals/"+testID.String()+"/schedule", validBody)
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "09:00")
	})

	t.Run("invalid uuid", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{})
		app := newProtectedApp()
		app.Put("/professionals/:id/schedule", h.SetSchedule)

		resp := doJSON(app, "PUT", "/professionals/bad-uuid/schedule", validBody)
		assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid json body", func(t *testing.T) {
		h := newProfHandler(&mockProfessionalSvc{})
		app := newProtectedApp()
		app.Put("/professionals/:id/schedule", h.SetSchedule)

		resp := doJSON(app, "PUT", "/professionals/"+testID.String()+"/schedule", `{not an array}`)
		assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	})
}
