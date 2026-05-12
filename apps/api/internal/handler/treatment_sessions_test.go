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

func setupTreatmentSessionApp(svc *mockTreatmentSessionSvc) (*handler.TreatmentSessionHandler, func(string, string) *http.Response, func(string, string, string) *http.Response) {
	h := handler.NewTreatmentSessionHandler(svc)
	app := newProtectedApp()
	app.Get("/treatments/:id/sessions", h.List)
	app.Post("/treatments/:id/sessions", h.Create)
	app.Get("/treatments/:id/sessions/:sid", h.GetByID)
	app.Patch("/treatments/:id/sessions/:sid", h.Update)
	app.Delete("/treatments/:id/sessions/:sid", h.Delete)

	doGet := func(method, path string) *http.Response {
		return doRequest(app, method, path)
	}
	doBody := func(method, path, body string) *http.Response {
		return doJSON(app, method, path, body)
	}
	return h, doGet, doBody
}

func TestTreatmentSessionHandler_List_Success(t *testing.T) {
	treatmentID := testID.String()
	svc := &mockTreatmentSessionSvc{
		listFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]*domain.TreatmentSession, error) {
			return []*domain.TreatmentSession{
				{ID: uuid.New(), Status: "pending"},
			}, nil
		},
	}
	_, doGet, _ := setupTreatmentSessionApp(svc)

	resp := doGet("GET", fmt.Sprintf("/treatments/%s/sessions", treatmentID))
	assertStatus(t, http.StatusOK, resp.StatusCode)
	assertBodyContains(t, resp, "data")
}

func TestTreatmentSessionHandler_List_BadTreatmentID(t *testing.T) {
	svc := &mockTreatmentSessionSvc{}
	_, doGet, _ := setupTreatmentSessionApp(svc)

	resp := doGet("GET", "/treatments/not-a-uuid/sessions")
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "id de tratamiento inválido")
}

func TestTreatmentSessionHandler_Create_Success(t *testing.T) {
	treatmentID := testID.String()
	svc := &mockTreatmentSessionSvc{
		createFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, input *domain.TreatmentSessionInput) (*domain.TreatmentSession, error) {
			return &domain.TreatmentSession{ID: uuid.New(), Status: input.Status}, nil
		},
	}
	_, _, doBody := setupTreatmentSessionApp(svc)

	body := `{"professional_id":"22222222-2222-2222-2222-222222222222","status":"pending","scheduled_at":"2025-06-01T10:00:00Z"}`
	resp := doBody("POST", fmt.Sprintf("/treatments/%s/sessions", treatmentID), body)
	assertStatus(t, http.StatusCreated, resp.StatusCode)
	assertBodyContains(t, resp, "pending")
}

func TestTreatmentSessionHandler_Create_BadBody(t *testing.T) {
	treatmentID := testID.String()
	svc := &mockTreatmentSessionSvc{}
	_, _, doBody := setupTreatmentSessionApp(svc)

	resp := doBody("POST", fmt.Sprintf("/treatments/%s/sessions", treatmentID), `{"status":"invalid_value"}`)
	assertStatus(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assertBodyContains(t, resp, "error")
}

func TestTreatmentSessionHandler_GetByID_Success(t *testing.T) {
	treatmentID := testID.String()
	sessionID := uuid.New().String()
	svc := &mockTreatmentSessionSvc{
		getByIDFn: func(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.TreatmentSession, error) {
			return &domain.TreatmentSession{ID: id, Status: "completed"}, nil
		},
	}
	_, doGet, _ := setupTreatmentSessionApp(svc)

	resp := doGet("GET", fmt.Sprintf("/treatments/%s/sessions/%s", treatmentID, sessionID))
	assertStatus(t, http.StatusOK, resp.StatusCode)
	assertBodyContains(t, resp, "completed")
}

func TestTreatmentSessionHandler_GetByID_BadSessionID(t *testing.T) {
	treatmentID := testID.String()
	svc := &mockTreatmentSessionSvc{}
	_, doGet, _ := setupTreatmentSessionApp(svc)

	resp := doGet("GET", fmt.Sprintf("/treatments/%s/sessions/bad-uuid", treatmentID))
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "id de sesión inválido")
}

func TestTreatmentSessionHandler_Update_Success(t *testing.T) {
	treatmentID := testID.String()
	sessionID := uuid.New().String()
	svc := &mockTreatmentSessionSvc{
		updateFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, id uuid.UUID, input *domain.UpdateTreatmentSessionInput) (*domain.TreatmentSession, error) {
			return &domain.TreatmentSession{ID: id, Status: input.Status}, nil
		},
	}
	_, _, doBody := setupTreatmentSessionApp(svc)

	resp := doBody("PATCH", fmt.Sprintf("/treatments/%s/sessions/%s", treatmentID, sessionID), `{"status":"completed"}`)
	assertStatus(t, http.StatusOK, resp.StatusCode)
	assertBodyContains(t, resp, "completed")
}

func TestTreatmentSessionHandler_Update_BadBody(t *testing.T) {
	treatmentID := testID.String()
	sessionID := uuid.New().String()
	svc := &mockTreatmentSessionSvc{}
	_, _, doBody := setupTreatmentSessionApp(svc)

	resp := doBody("PATCH", fmt.Sprintf("/treatments/%s/sessions/%s", treatmentID, sessionID), `{"status":"nope"}`)
	assertStatus(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assertBodyContains(t, resp, "error")
}

func TestTreatmentSessionHandler_Delete_Success(t *testing.T) {
	treatmentID := testID.String()
	sessionID := uuid.New().String()
	svc := &mockTreatmentSessionSvc{
		deleteFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) error {
			return nil
		},
	}
	_, doGet, _ := setupTreatmentSessionApp(svc)

	resp := doGet("DELETE", fmt.Sprintf("/treatments/%s/sessions/%s", treatmentID, sessionID))
	assertStatus(t, http.StatusNoContent, resp.StatusCode)
}

func TestTreatmentSessionHandler_Delete_BadSessionID(t *testing.T) {
	treatmentID := testID.String()
	svc := &mockTreatmentSessionSvc{}
	_, doGet, _ := setupTreatmentSessionApp(svc)

	resp := doGet("DELETE", fmt.Sprintf("/treatments/%s/sessions/not-valid", treatmentID))
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "id de sesión inválido")
}
