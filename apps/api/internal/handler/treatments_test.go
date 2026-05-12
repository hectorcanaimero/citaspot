package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

// validTreatmentBody retorna un JSON valido para TreatmentInput.
func validTreatmentBody() string {
	return fmt.Sprintf(
		`{"customer_id":"%s","professional_id":"%s","name":"Ortodoncia superior","treatment_type":"ortodoncia"}`,
		testID, testTenantID,
	)
}

func TestTreatmentHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		mockFn     func(context.Context, uuid.UUID, *domain.TreatmentListQuery) ([]*domain.Treatment, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success empty list",
			query:      "",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:  "success with treatments",
			query: "",
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
				return []*domain.Treatment{
					{ID: testID, Name: "Ortodoncia superior", TreatmentType: "ortodoncia", Status: "proposed"},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "Ortodoncia superior",
		},
		{
			name:  "filters by status and customer_id",
			query: fmt.Sprintf("?status=in_progress&customer_id=%s", testID),
			mockFn: func(_ context.Context, _ uuid.UUID, q *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
				return []*domain.Treatment{}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:  "service error returns 500",
			query: "",
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewTreatmentHandler(&mockTreatmentSvc{listFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/treatments", h.List)

			resp := doRequest(app, "GET", "/treatments"+tc.query)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTreatmentHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID, *domain.TreatmentInput) (*domain.Treatment, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       validTreatmentBody(),
			wantStatus: http.StatusCreated,
			wantBody:   "Ortodoncia superior",
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing required fields",
			body:       `{"name":"Ortodoncia"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "name too short",
			body: fmt.Sprintf(
				`{"customer_id":"%s","professional_id":"%s","name":"A","treatment_type":"ortodoncia"}`,
				testID, testTenantID,
			),
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "invalid treatment_type",
			body: fmt.Sprintf(
				`{"customer_id":"%s","professional_id":"%s","name":"Tratamiento X","treatment_type":"inventado"}`,
				testID, testTenantID,
			),
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "service error returns 500",
			body: validTreatmentBody(),
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.TreatmentInput) (*domain.Treatment, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewTreatmentHandler(&mockTreatmentSvc{createFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/treatments", h.Create)

			resp := doJSON(app, "POST", "/treatments", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTreatmentHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Treatment, error)
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
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Treatment, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewTreatmentHandler(&mockTreatmentSvc{getByIDFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/treatments/:id", h.GetByID)

			resp := doRequest(app, "GET", "/treatments/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTreatmentHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.TreatmentInput) (*domain.Treatment, error)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			id:   testID.String(),
			body: validTreatmentBody(),
			mockFn: func(_ context.Context, _ uuid.UUID, id uuid.UUID, input *domain.TreatmentInput) (*domain.Treatment, error) {
				return &domain.Treatment{ID: id, Name: input.Name}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "Ortodoncia superior",
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       validTreatmentBody(),
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
			body: validTreatmentBody(),
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.TreatmentInput) (*domain.Treatment, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewTreatmentHandler(&mockTreatmentSvc{updateFn: tc.mockFn})
			app := newProtectedApp()
			app.Put("/treatments/:id", h.Update)

			resp := doJSON(app, "PUT", "/treatments/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTreatmentHandler_UpdateStatus(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.UpdateTreatmentStatusInput) (*domain.Treatment, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			body:       `{"status":"in_progress"}`,
			wantStatus: http.StatusOK,
			wantBody:   testID.String(),
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       `{"status":"completed"}`,
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
			name:       "invalid status value",
			id:         testID.String(),
			body:       `{"status":"inventado"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "missing status",
			id:         testID.String(),
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "not found",
			id:   testID.String(),
			body: `{"status":"completed"}`,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.UpdateTreatmentStatusInput) (*domain.Treatment, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewTreatmentHandler(&mockTreatmentSvc{updateStatusFn: tc.mockFn})
			app := newProtectedApp()
			app.Patch("/treatments/:id/status", h.UpdateStatus)

			resp := doJSON(app, "PATCH", "/treatments/"+tc.id+"/status", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
