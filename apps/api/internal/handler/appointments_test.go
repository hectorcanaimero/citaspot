package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

const (
	validProfID    = "44444444-4444-4444-4444-444444444444"
	validServiceID = "55555555-5555-5555-5555-555555555555"
)

func newApptHandler(apptSvc *mockAppointmentSvc, availSvc *mockAvailabilitySvc) *handler.AppointmentHandler {
	return handler.NewAppointmentHandler(apptSvc, availSvc)
}

func TestAppointmentHandler_Availability(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			query:      "?professional_id=" + validProfID + "&service_id=" + validServiceID + "&date=2026-03-20",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "missing professional_id",
			query:      "?service_id=" + validServiceID + "&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "professional_id",
		},
		{
			name:       "invalid professional_id",
			query:      "?professional_id=bad&service_id=" + validServiceID + "&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "professional_id",
		},
		{
			name:       "missing service_id",
			query:      "?professional_id=" + validProfID + "&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "service_id",
		},
		{
			name:       "invalid service_id",
			query:      "?professional_id=" + validProfID + "&service_id=bad&date=2026-03-20",
			wantStatus: http.StatusBadRequest,
			wantBody:   "service_id",
		},
		{
			name:       "missing date",
			query:      "?professional_id=" + validProfID + "&service_id=" + validServiceID,
			wantStatus: http.StatusBadRequest,
			wantBody:   "date",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newApptHandler(&mockAppointmentSvc{}, &mockAvailabilitySvc{})
			app := newProtectedApp()
			app.Get("/appointments/availability", h.Availability)

			resp := doRequest(app, "GET", "/appointments/availability"+tc.query)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestAppointmentHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		mockFn     func(context.Context, uuid.UUID, string, string) ([]*domain.AppointmentWithDetails, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success with date",
			query:      "?date=2026-03-20",
			wantStatus: http.StatusOK,
			wantBody:   "data",
		},
		{
			name:       "missing date parameter",
			query:      "",
			wantStatus: http.StatusBadRequest,
			wantBody:   "date",
		},
		{
			name:  "success with results",
			query: "?date=2026-03-20",
			mockFn: func(_ context.Context, _ uuid.UUID, date, _ string) ([]*domain.AppointmentWithDetails, error) {
				return []*domain.AppointmentWithDetails{
					{CustomerName: "Ana García"},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   "Ana García",
		},
		{
			name:  "service error returns 500",
			query: "?date=2026-03-20",
			mockFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]*domain.AppointmentWithDetails, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newApptHandler(&mockAppointmentSvc{listFn: tc.mockFn}, &mockAvailabilitySvc{})
			app := newProtectedApp()
			app.Get("/appointments", h.List)

			resp := doRequest(app, "GET", "/appointments"+tc.query)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestAppointmentHandler_Create(t *testing.T) {
	validBody := `{
		"professional_id":"` + validProfID + `",
		"service_id":"` + validServiceID + `",
		"starts_at":"2026-03-20T10:00:00Z",
		"customer_name":"Ana García",
		"customer_phone":"+18091234567"
	}`

	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID, *domain.CreateAppointmentRequest) (*domain.Appointment, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       validBody,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "missing professional_id",
			body:       `{"service_id":"` + validServiceID + `","starts_at":"2026-03-20T10:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name:       "missing service_id",
			body:       `{"professional_id":"` + validProfID + `","starts_at":"2026-03-20T10:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "slot unavailable",
			body: validBody,
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
				return nil, domain.ErrSlotUnavailable
			},
			wantStatus: http.StatusConflict,
			wantBody:   "horario",
		},
		{
			name: "conflict",
			body: validBody,
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
				return nil, domain.ErrConflict
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newApptHandler(&mockAppointmentSvc{createFn: tc.mockFn}, &mockAvailabilitySvc{})
			app := newProtectedApp()
			app.Post("/appointments", h.Create)

			resp := doJSON(app, "POST", "/appointments", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

// TestAppointmentHandler_Create_WithTreatmentID verifica que cuando el body
// incluye treatment_id, el campo se propaga al service para materializar la
// treatment_session vinculada (lógica de Plane #32 / migration 037).
func TestAppointmentHandler_Create_WithTreatmentID(t *testing.T) {
	treatmentID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	body := `{
		"professional_id":"` + validProfID + `",
		"service_id":"` + validServiceID + `",
		"starts_at":"2026-03-20T10:00:00Z",
		"customer_name":"Ana García",
		"customer_phone":"+18091234567",
		"treatment_id":"` + treatmentID.String() + `"
	}`

	var captured *domain.CreateAppointmentRequest
	mock := &mockAppointmentSvc{
		createFn: func(_ context.Context, _ uuid.UUID, req *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
			captured = req
			return &domain.Appointment{ID: uuid.New(), TreatmentID: req.TreatmentID}, nil
		},
	}

	h := newApptHandler(mock, &mockAvailabilitySvc{})
	app := newProtectedApp()
	app.Post("/appointments", h.Create)

	resp := doJSON(app, "POST", "/appointments", body)
	assertStatus(t, http.StatusCreated, resp.StatusCode)

	if captured == nil {
		t.Fatal("service Create no fue invocado")
	}
	if captured.TreatmentID == nil {
		t.Fatal("TreatmentID no fue propagado al service")
	}
	if *captured.TreatmentID != treatmentID {
		t.Fatalf("TreatmentID esperado %s, recibido %s", treatmentID, *captured.TreatmentID)
	}
}

// TestAppointmentHandler_Create_WithoutTreatmentID verifica que un appointment
// sin treatment_id mantiene TreatmentID = nil (no materializa session).
func TestAppointmentHandler_Create_WithoutTreatmentID(t *testing.T) {
	body := `{
		"professional_id":"` + validProfID + `",
		"service_id":"` + validServiceID + `",
		"starts_at":"2026-03-20T10:00:00Z",
		"customer_name":"Ana García",
		"customer_phone":"+18091234567"
	}`

	var captured *domain.CreateAppointmentRequest
	mock := &mockAppointmentSvc{
		createFn: func(_ context.Context, _ uuid.UUID, req *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
			captured = req
			return &domain.Appointment{ID: uuid.New()}, nil
		},
	}

	h := newApptHandler(mock, &mockAvailabilitySvc{})
	app := newProtectedApp()
	app.Post("/appointments", h.Create)

	resp := doJSON(app, "POST", "/appointments", body)
	assertStatus(t, http.StatusCreated, resp.StatusCode)

	if captured == nil {
		t.Fatal("service Create no fue invocado")
	}
	if captured.TreatmentID != nil {
		t.Fatalf("TreatmentID debería ser nil, recibido %s", *captured.TreatmentID)
	}
}

func TestAppointmentHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.AppointmentWithDetails, error)
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
			id:         "not-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name: "not found",
			id:   testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.AppointmentWithDetails, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newApptHandler(&mockAppointmentSvc{getByIDFn: tc.mockFn}, &mockAvailabilitySvc{})
			app := newProtectedApp()
			app.Get("/appointments/:id", h.GetByID)

			resp := doRequest(app, "GET", "/appointments/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestAppointmentHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.UpdateAppointmentRequest) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success confirmed",
			id:         testID.String(),
			body:       `{"status":"confirmed"}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "success no_show",
			id:         testID.String(),
			body:       `{"status":"no_show"}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       `{"status":"confirmed"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name:       "invalid json",
			id:         testID.String(),
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "invalid status value",
			id:         testID.String(),
			body:       `{"status":"invalid_status"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "inválidos",
		},
		{
			name: "not found",
			id:   testID.String(),
			body: `{"status":"confirmed"}`,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.UpdateAppointmentRequest) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newApptHandler(&mockAppointmentSvc{updateFn: tc.mockFn}, &mockAvailabilitySvc{})
			app := newProtectedApp()
			app.Patch("/appointments/:id", h.Update)

			resp := doJSON(app, "PATCH", "/appointments/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestAppointmentHandler_Cancel(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, string) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success with reason",
			id:         testID.String(),
			body:       `{"reason":"Cliente canceló"}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "success without reason",
			id:         testID.String(),
			body:       `{}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name: "not found",
			id:   testID.String(),
			body: `{}`,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ string) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newApptHandler(&mockAppointmentSvc{cancelFn: tc.mockFn}, &mockAvailabilitySvc{})
			app := newProtectedApp()
			app.Delete("/appointments/:id/cancel", h.Cancel)

			resp := doJSON(app, "DELETE", "/appointments/"+tc.id+"/cancel", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
