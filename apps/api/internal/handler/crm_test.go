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

func TestCRMHandler_Metrics_Success(t *testing.T) {
	repo := &mockCRMMetricsRepo{
		getMetricsFn: func(_ context.Context, _ uuid.UUID) (*domain.CRMMetrics, error) {
			return &domain.CRMMetrics{
				ActiveTreatments: 3,
				PendingTasks:     5,
			}, nil
		},
	}
	h := handler.NewCRMHandler(repo)
	app := newProtectedApp()
	app.Get("/crm/metrics", h.Metrics)

	resp := doRequest(app, "GET", "/crm/metrics")
	assertStatus(t, http.StatusOK, resp.StatusCode)
	assertBodyContains(t, resp, `"active_treatments":3`)
}

func TestCRMHandler_Metrics_RepoError(t *testing.T) {
	repo := &mockCRMMetricsRepo{
		getMetricsFn: func(_ context.Context, _ uuid.UUID) (*domain.CRMMetrics, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}
	h := handler.NewCRMHandler(repo)
	app := newProtectedApp()
	app.Get("/crm/metrics", h.Metrics)

	resp := doRequest(app, "GET", "/crm/metrics")
	assertStatus(t, http.StatusInternalServerError, resp.StatusCode)
}
