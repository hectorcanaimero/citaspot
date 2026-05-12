package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newRuleHandler(svc *mockRuleSvc) *handler.RuleHandler {
	return handler.NewRuleHandler(svc)
}

const validRuleBody = `{
	"name":"Recordatorio post-cita",
	"trigger_type":"event",
	"trigger_event":"appointment.completed",
	"actions":[{"type":"send_whatsapp","template":"Gracias por tu visita"}]
}`

func TestRuleHandler_List(t *testing.T) {
	t.Run("success empty list", func(t *testing.T) {
		h := newRuleHandler(&mockRuleSvc{})
		app := newProtectedApp()
		app.Get("/rules", h.List)

		resp := doRequest(app, "GET", "/rules")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "data")
	})

	t.Run("success with rules", func(t *testing.T) {
		h := newRuleHandler(&mockRuleSvc{
			listFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Rule, error) {
				return []*domain.Rule{
					{ID: testID, Name: "Bienvenida nuevo cliente"},
				}, nil
			},
		})
		app := newProtectedApp()
		app.Get("/rules", h.List)

		resp := doRequest(app, "GET", "/rules")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "Bienvenida nuevo cliente")
	})

	t.Run("service error returns 500", func(t *testing.T) {
		h := newRuleHandler(&mockRuleSvc{
			listFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Rule, error) {
				return nil, domain.ErrInternal
			},
		})
		app := newProtectedApp()
		app.Get("/rules", h.List)

		resp := doRequest(app, "GET", "/rules")
		assertStatus(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestRuleHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID, *domain.RuleInput) (*domain.Rule, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       validRuleBody,
			wantStatus: http.StatusCreated,
			wantBody:   "Recordatorio post-cita",
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing name",
			body:       `{"trigger_type":"event","actions":[{"type":"send_whatsapp"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid trigger_type",
			body:       `{"name":"Test Rule","trigger_type":"invalid","actions":[{"type":"send_whatsapp"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing actions",
			body:       `{"name":"Test Rule","trigger_type":"event"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty actions array",
			body:       `{"name":"Test Rule","trigger_type":"event","actions":[]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: validRuleBody,
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.RuleInput) (*domain.Rule, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newRuleHandler(&mockRuleSvc{createFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/rules", h.Create)

			resp := doJSON(app, "POST", "/rules", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestRuleHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Rule, error)
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
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Rule, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newRuleHandler(&mockRuleSvc{getByIDFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/rules/:id", h.GetByID)

			resp := doRequest(app, "GET", "/rules/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestRuleHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.RuleInput) (*domain.Rule, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "success",
			id:     testID.String(),
			body:   validRuleBody,
			mockFn: func(_ context.Context, _ uuid.UUID, id uuid.UUID, input *domain.RuleInput) (*domain.Rule, error) {
				return &domain.Rule{ID: id, Name: input.Name}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "Recordatorio post-cita",
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       validRuleBody,
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name:       "invalid json",
			id:         testID.String(),
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			id:   testID.String(),
			body: validRuleBody,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.RuleInput) (*domain.Rule, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newRuleHandler(&mockRuleSvc{updateFn: tc.mockFn})
			app := newProtectedApp()
			app.Put("/rules/:id", h.Update)

			resp := doJSON(app, "PUT", "/rules/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestRuleHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name: "not found",
			id:   testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newRuleHandler(&mockRuleSvc{deleteFn: tc.mockFn})
			app := newProtectedApp()
			app.Delete("/rules/:id", h.Delete)

			resp := doRequest(app, "DELETE", "/rules/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestRuleHandler_ListExecutions(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		query      string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, int) ([]*domain.RuleExecution, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success empty list",
			id:         testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "with custom limit",
			id:         testID.String(),
			query:      "?limit=5",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "invalid uuid",
			id:         "not-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name: "not found",
			id:   testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ int) ([]*domain.RuleExecution, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newRuleHandler(&mockRuleSvc{listExecutionsFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/rules/:id/executions", h.ListExecutions)

			path := "/rules/" + tc.id + "/executions" + tc.query
			resp := doRequest(app, "GET", path)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
