package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func TestBrandingHandler_Remove(t *testing.T) {
	tests := []struct {
		name       string
		kind       string
		mockFn     func(context.Context, uuid.UUID, domain.BrandingAssetKind) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success remove logo",
			kind:       "logo",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "success remove cover",
			kind:       "cover",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid kind",
			kind:       "banner",
			wantStatus: http.StatusBadRequest,
			wantBody:   "kind",
		},
		{
			name: "not found",
			kind: "logo",
			mockFn: func(_ context.Context, _ uuid.UUID, _ domain.BrandingAssetKind) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "service error",
			kind: "cover",
			mockFn: func(_ context.Context, _ uuid.UUID, _ domain.BrandingAssetKind) error {
				return domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewBrandingHandler(&mockBrandingSvc{removeAssetFn: tc.mockFn})
			app := newProtectedApp()
			app.Delete("/tenant/branding/:kind", h.Remove)

			resp := doRequest(app, "DELETE", "/tenant/branding/"+tc.kind)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestBrandingHandler_Remove_NilService(t *testing.T) {
	h := handler.NewBrandingHandler(nil)
	app := newProtectedApp()
	app.Delete("/tenant/branding/:kind", h.Remove)

	resp := doRequest(app, "DELETE", "/tenant/branding/logo")
	assertStatus(t, http.StatusServiceUnavailable, resp.StatusCode)
	assertBodyContains(t, resp, "MINIO")
}
