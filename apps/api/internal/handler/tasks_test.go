package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newTaskHandler(svc *mockTaskSvc) *handler.TaskHandler {
	return handler.NewTaskHandler(svc)
}

const validTaskBody = `{"title":"Llamar paciente para confirmar cita"}`

func TestTaskHandler_List(t *testing.T) {
	t.Run("success empty list", func(t *testing.T) {
		h := newTaskHandler(&mockTaskSvc{})
		app := newProtectedApp()
		app.Get("/tasks", h.List)

		resp := doRequest(app, "GET", "/tasks")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "data")
	})

	t.Run("success with tasks", func(t *testing.T) {
		h := newTaskHandler(&mockTaskSvc{
			listFn: func(_ context.Context, _ uuid.UUID, _ *domain.TaskListQuery) ([]*domain.Task, error) {
				return []*domain.Task{
					{ID: testID, Title: "Seguimiento tratamiento"},
				}, nil
			},
		})
		app := newProtectedApp()
		app.Get("/tasks", h.List)

		resp := doRequest(app, "GET", "/tasks")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "Seguimiento tratamiento")
	})

	t.Run("with query filters", func(t *testing.T) {
		h := newTaskHandler(&mockTaskSvc{})
		app := newProtectedApp()
		app.Get("/tasks", h.List)

		resp := doRequest(app, "GET", "/tasks?status=pending&assigned_to="+testUserID.String()+"&customer_id="+testID.String()+"&due_before=2026-06-01T00:00:00Z")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "data")
	})

	t.Run("service error returns 500", func(t *testing.T) {
		h := newTaskHandler(&mockTaskSvc{
			listFn: func(_ context.Context, _ uuid.UUID, _ *domain.TaskListQuery) ([]*domain.Task, error) {
				return nil, domain.ErrInternal
			},
		})
		app := newProtectedApp()
		app.Get("/tasks", h.List)

		resp := doRequest(app, "GET", "/tasks")
		assertStatus(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestTaskHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID, *domain.TaskInput) (*domain.Task, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       validTaskBody,
			wantStatus: http.StatusCreated,
			wantBody:   "Llamar paciente",
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing title",
			body:       `{"description":"sin titulo"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "title too short",
			body:       `{"title":"A"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: validTaskBody,
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.TaskInput) (*domain.Task, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTaskHandler(&mockTaskSvc{createFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/tasks", h.Create)

			resp := doJSON(app, "POST", "/tasks", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTaskHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Task, error)
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
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Task, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTaskHandler(&mockTaskSvc{getByIDFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/tasks/:id", h.GetByID)

			resp := doRequest(app, "GET", "/tasks/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTaskHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.TaskInput) (*domain.Task, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			body:       `{"title":"Tarea actualizada"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       validTaskBody,
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
			body: validTaskBody,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.TaskInput) (*domain.Task, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTaskHandler(&mockTaskSvc{updateFn: tc.mockFn})
			app := newProtectedApp()
			app.Put("/tasks/:id", h.Update)

			resp := doJSON(app, "PUT", "/tasks/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTaskHandler_Complete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Task, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   "completed",
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
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Task, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTaskHandler(&mockTaskSvc{completeFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/tasks/:id/complete", h.Complete)

			resp := doRequest(app, "POST", "/tasks/"+tc.id+"/complete")
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestTaskHandler_Dismiss(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Task, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   "dismissed",
		},
		{
			name:       "invalid uuid",
			id:         "bad",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTaskHandler(&mockTaskSvc{dismissFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/tasks/:id/dismiss", h.Dismiss)

			resp := doRequest(app, "POST", "/tasks/"+tc.id+"/dismiss")
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
