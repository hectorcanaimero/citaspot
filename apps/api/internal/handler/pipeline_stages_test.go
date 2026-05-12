package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

// setupStageApp crea un Fiber app protegido con las rutas del handler registradas.
func setupStageApp(svc *mockPipelineStageSvc) *handler.PipelineStageHandler {
	return handler.NewPipelineStageHandler(svc)
}

func TestPipelineStageHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		mockFn     func(context.Context, uuid.UUID) ([]*domain.PipelineStage, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success empty list",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name: "success with stages",
			mockFn: func(_ context.Context, _ uuid.UUID) ([]*domain.PipelineStage, error) {
				return []*domain.PipelineStage{
					{ID: testID, Name: "Nuevo Lead", Position: 0},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "Nuevo Lead",
		},
		{
			name: "service error returns 500",
			mockFn: func(_ context.Context, _ uuid.UUID) ([]*domain.PipelineStage, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPipelineStageSvc{listFn: tc.mockFn}
			h := setupStageApp(svc)
			app := newProtectedApp()
			app.Get("/stages", h.List)

			resp := doRequest(app, "GET", "/stages")
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestPipelineStageHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       `{"name":"Contactado","position":1,"color":"#FF5733"}`,
			wantStatus: http.StatusCreated,
			wantBody:   "Contactado",
		},
		{
			name:       "invalid json",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON inválido",
		},
		{
			name:       "name too short",
			body:       `{"name":"A"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Datos inválidos",
		},
		{
			name:       "missing name",
			body:       `{"position":1}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Datos inválidos",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPipelineStageSvc{}
			h := setupStageApp(svc)
			app := newProtectedApp()
			app.Post("/stages", h.Create)

			resp := doJSON(app, "POST", "/stages", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestPipelineStageHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   "Test Stage",
		},
		{
			name:       "invalid uuid",
			id:         "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID inválido",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPipelineStageSvc{}
			h := setupStageApp(svc)
			app := newProtectedApp()
			app.Get("/stages/:id", h.GetByID)

			resp := doRequest(app, "GET", "/stages/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestPipelineStageHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			body:       `{"name":"Actualizado"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid",
			id:         "bad-id",
			body:       `{"name":"Algo"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID inválido",
		},
		{
			name:       "invalid json",
			id:         testID.String(),
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON inválido",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPipelineStageSvc{}
			h := setupStageApp(svc)
			app := newProtectedApp()
			app.Put("/stages/:id", h.Update)

			resp := doJSON(app, "PUT", "/stages/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestPipelineStageHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
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
			id:         "nope",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID inválido",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPipelineStageSvc{}
			h := setupStageApp(svc)
			app := newProtectedApp()
			app.Delete("/stages/:id", h.Delete)

			resp := doRequest(app, "DELETE", "/stages/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestPipelineStageHandler_Reorder(t *testing.T) {
	stageA := uuid.New()
	stageB := uuid.New()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       `[{"stage_id":"` + stageA.String() + `","position":0},{"stage_id":"` + stageB.String() + `","position":1}]`,
			wantStatus: http.StatusOK,
			wantBody:   `"ok":true`,
		},
		{
			name:       "invalid json",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON inválido",
		},
		{
			name:       "empty array",
			body:       `[]`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Lista de etapas vacía",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPipelineStageSvc{}
			h := setupStageApp(svc)
			app := newProtectedApp()
			// Reorder debe registrarse ANTES de :id para no ser capturado como param.
			app.Put("/stages/reorder", h.Reorder)

			resp := doJSON(app, "PUT", "/stages/reorder", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
