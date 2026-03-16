package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func TestCustomerHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		mockFn     func(context.Context, uuid.UUID, string, int, int) ([]*domain.Customer, error)
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
			name:  "success with customers",
			query: "",
			mockFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*domain.Customer, error) {
				return []*domain.Customer{
					{ID: testID, Name: "María López", Phone: "+18099876543"},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "María López",
		},
		{
			name:       "with search query",
			query:      "?search=María",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "with pagination",
			query:      "?limit=10&offset=20",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "limit out of range uses default 50",
			query:      "?limit=999",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:  "repository error returns 500",
			query: "",
			mockFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*domain.Customer, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewCustomerHandler(&mockCustomerRepo{listFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/customers", h.List)

			resp := doRequest(app, "GET", "/customers"+tc.query)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
