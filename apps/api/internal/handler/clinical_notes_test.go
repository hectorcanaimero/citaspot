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

func validClinicalNoteBody() string {
	return fmt.Sprintf(
		`{"professional_id":"%s","subjective":"Dolor de cabeza","objective":"Tensión muscular","assessment":"Cefalea tensional","plan":"Ibuprofeno 400mg"}`,
		testID,
	)
}

func TestClinicalNoteHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		apptID     string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.CreateClinicalNoteRequest) (*domain.ClinicalNote, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			apptID:     testID.String(),
			body:       validClinicalNoteBody(),
			wantStatus: http.StatusCreated,
			wantBody:   "appointment_id",
		},
		{
			name:       "invalid appointment uuid",
			apptID:     "bad-uuid",
			body:       validClinicalNoteBody(),
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:       "invalid json",
			apptID:     testID.String(),
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing required professional_id",
			apptID:     testID.String(),
			body:       `{"subjective":"dolor"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "service error not found",
			apptID: testID.String(),
			body:   validClinicalNoteBody(),
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.CreateClinicalNoteRequest) (*domain.ClinicalNote, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "service error clinical note exists",
			apptID: testID.String(),
			body:   validClinicalNoteBody(),
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.CreateClinicalNoteRequest) (*domain.ClinicalNote, error) {
				return nil, domain.ErrClinicalNoteExists
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalNoteHandler(&mockClinicalNoteSvc{createFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/appointments/:id/clinical-note", h.Create)

			resp := doJSON(app, "POST", "/appointments/"+tc.apptID+"/clinical-note", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestClinicalNoteHandler_GetByAppointment(t *testing.T) {
	tests := []struct {
		name       string
		apptID     string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.ClinicalNoteWithDetails, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			apptID:     testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   "appointment_id",
		},
		{
			name:       "invalid uuid",
			apptID:     "bad",
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:   "not found",
			apptID: testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalNoteHandler(&mockClinicalNoteSvc{getByAppointmentFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/appointments/:id/clinical-note", h.GetByAppointment)

			resp := doRequest(app, "GET", "/appointments/"+tc.apptID+"/clinical-note")
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestClinicalNoteHandler_ListByCustomer(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		query      string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*domain.ClinicalNoteWithDetails, int, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success empty",
			customerID: testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "with limit and offset",
			customerID: testID.String(),
			query:      "?limit=5&offset=10",
			wantStatus: http.StatusOK,
			wantBody:   "total",
		},
		{
			name:       "invalid customer uuid",
			customerID: "not-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:       "negative limit defaults",
			customerID: testID.String(),
			query:      "?limit=-1&offset=-5",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "over 100 limit defaults to 20",
			customerID: testID.String(),
			query:      "?limit=200",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "service error",
			customerID: testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID, _, _ int) ([]*domain.ClinicalNoteWithDetails, int, error) {
				return nil, 0, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalNoteHandler(&mockClinicalNoteSvc{listByCustomerFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/customers/:id/clinical-notes", h.ListByCustomer)

			resp := doRequest(app, "GET", "/customers/"+tc.customerID+"/clinical-notes"+tc.query)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestClinicalNoteHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		noteID     string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.ClinicalNoteWithDetails, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			noteID:     testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   testID.String(),
		},
		{
			name:       "invalid uuid",
			noteID:     "bad",
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:   "not found",
			noteID: testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalNoteHandler(&mockClinicalNoteSvc{getByIDFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/customers/:id/clinical-notes/:noteId", h.GetByID)

			resp := doRequest(app, "GET", "/customers/"+testID.String()+"/clinical-notes/"+tc.noteID)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestClinicalNoteHandler_Update(t *testing.T) {
	subj := "Actualizado"
	tests := []struct {
		name       string
		noteID     string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.UpdateClinicalNoteRequest) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			noteID:     testID.String(),
			body:       fmt.Sprintf(`{"subjective":"%s"}`, subj),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			noteID:     "bad",
			body:       `{"subjective":"x"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:       "invalid json",
			noteID:     testID.String(),
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:   "not found",
			noteID: testID.String(),
			body:   `{"subjective":"x"}`,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.UpdateClinicalNoteRequest) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalNoteHandler(&mockClinicalNoteSvc{updateFn: tc.mockFn})
			app := newProtectedApp()
			app.Patch("/clinical-notes/:noteId", h.Update)

			resp := doJSON(app, "PATCH", "/clinical-notes/"+tc.noteID, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestClinicalNoteHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		noteID     string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			noteID:     testID.String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			noteID:     "bad",
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:   "not found",
			noteID: testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalNoteHandler(&mockClinicalNoteSvc{deleteFn: tc.mockFn})
			app := newProtectedApp()
			app.Delete("/clinical-notes/:noteId", h.Delete)

			resp := doRequest(app, "DELETE", "/clinical-notes/"+tc.noteID)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
