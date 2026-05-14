package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func TestClinicalFileHandler_ListByNote(t *testing.T) {
	tests := []struct {
		name       string
		noteID     string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) ([]*domain.ClinicalFile, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success empty",
			noteID:     testID.String(),
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:   "success with files",
			noteID: testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) ([]*domain.ClinicalFile, error) {
				return []*domain.ClinicalFile{
					{ID: uuid.New(), FileName: "radiografia.jpg"},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "radiografia.jpg",
		},
		{
			name:       "invalid note uuid",
			noteID:     "bad-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:   "service error",
			noteID: testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) ([]*domain.ClinicalFile, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalFileHandler(&mockClinicalFileSvc{listByNoteFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/clinical-notes/:noteId/files", h.ListByNote)

			resp := doRequest(app, "GET", "/clinical-notes/"+tc.noteID+"/files")
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestClinicalFileHandler_ListByCustomer(t *testing.T) {
	tests := []struct {
		name       string
		customerID string
		query      string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, string, int, int) ([]*domain.ClinicalFile, int, error)
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
			name:       "with category filter",
			customerID: testID.String(),
			query:      "?category=radiography&limit=10&offset=0",
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
			name:       "negative limit and offset defaults",
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
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ string, _, _ int) ([]*domain.ClinicalFile, int, error) {
				return nil, 0, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalFileHandler(&mockClinicalFileSvc{listByCustomerFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/customers/:id/clinical-files", h.ListByCustomer)

			resp := doRequest(app, "GET", "/customers/"+tc.customerID+"/clinical-files"+tc.query)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestClinicalFileHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		fileID     string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			fileID:     testID.String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			fileID:     "bad",
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválido",
		},
		{
			name:   "not found",
			fileID: testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewClinicalFileHandler(&mockClinicalFileSvc{deleteFn: tc.mockFn})
			app := newProtectedApp()
			app.Delete("/clinical-files/:fileId", h.Delete)

			resp := doRequest(app, "DELETE", "/clinical-files/"+tc.fileID)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
