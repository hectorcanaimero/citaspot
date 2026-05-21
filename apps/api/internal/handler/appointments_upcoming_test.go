package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/google/uuid"
)

// TestAppointmentHandler_Upcoming_InvalidLimit verifica que el handler
// retorna 400 cuando ?limit no es un entero válido (sin alcanzar el service).
func TestAppointmentHandler_Upcoming_InvalidLimit(t *testing.T) {
	called := false
	svc := &mockAppointmentSvc{
		listUpcomingFn: func(_ context.Context, _ uuid.UUID, _ int) ([]*domain.AppointmentWithDetails, error) {
			called = true
			return nil, nil
		},
	}
	h := newApptHandler(svc, &mockAvailabilitySvc{})
	app := newProtectedApp()
	app.Get("/appointments/upcoming", h.Upcoming)

	resp := doRequest(app, "GET", "/appointments/upcoming?limit=abc")
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "limit")

	if called {
		t.Fatal("service.ListUpcoming no debería invocarse con limit inválido")
	}
}

// TestAppointmentHandler_Upcoming_DefaultLimit verifica que sin ?limit
// el handler pasa 10 al service (default).
func TestAppointmentHandler_Upcoming_DefaultLimit(t *testing.T) {
	gotLimit := -1
	svc := &mockAppointmentSvc{
		listUpcomingFn: func(_ context.Context, _ uuid.UUID, limit int) ([]*domain.AppointmentWithDetails, error) {
			gotLimit = limit
			return []*domain.AppointmentWithDetails{}, nil
		},
	}
	h := newApptHandler(svc, &mockAvailabilitySvc{})
	app := newProtectedApp()
	app.Get("/appointments/upcoming", h.Upcoming)

	resp := doRequest(app, "GET", "/appointments/upcoming")
	assertStatus(t, http.StatusOK, resp.StatusCode)
	if gotLimit != 10 {
		t.Errorf("default limit esperado 10, got %d", gotLimit)
	}
}

// TestAppointmentHandler_Upcoming_PassesLimit verifica que el handler
// reenvía el limit del query string tal cual al service (el clamping lo
// hace el service, no el handler).
func TestAppointmentHandler_Upcoming_PassesLimit(t *testing.T) {
	gotLimit := -1
	svc := &mockAppointmentSvc{
		listUpcomingFn: func(_ context.Context, _ uuid.UUID, limit int) ([]*domain.AppointmentWithDetails, error) {
			gotLimit = limit
			return []*domain.AppointmentWithDetails{}, nil
		},
	}
	h := newApptHandler(svc, &mockAvailabilitySvc{})
	app := newProtectedApp()
	app.Get("/appointments/upcoming", h.Upcoming)

	resp := doRequest(app, "GET", "/appointments/upcoming?limit=25")
	assertStatus(t, http.StatusOK, resp.StatusCode)
	if gotLimit != 25 {
		t.Errorf("limit esperado 25, got %d", gotLimit)
	}
}
