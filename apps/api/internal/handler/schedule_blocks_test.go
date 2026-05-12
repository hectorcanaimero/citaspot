package handler_test

import (
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/handler"
)

func newScheduleBlockApp(repo *mockScheduleRepo) (func(string, string, string) *http.Response, func(string, string) *http.Response) {
	h := handler.NewScheduleBlockHandler(repo)
	app := newProtectedApp()
	app.Post("/blocks", h.Create)
	app.Get("/blocks", h.List)
	app.Delete("/blocks/:id", h.Delete)
	return func(method, path, body string) *http.Response {
			return doJSON(app, method, path, body)
		}, func(method, path string) *http.Response {
			return doRequest(app, method, path)
		}
}

func TestScheduleBlockHandler_Create_NonRecurring_Success(t *testing.T) {
	doJ, _ := newScheduleBlockApp(&mockScheduleRepo{})

	body := `{
		"professional_id": "33333333-3333-3333-3333-333333333333",
		"starts_at": "2026-06-01T09:00:00Z",
		"ends_at": "2026-06-01T12:00:00Z",
		"reason": "Vacaciones"
	}`
	resp := doJ("POST", "/blocks", body)
	assertStatus(t, http.StatusCreated, resp.StatusCode)
	assertBodyContains(t, resp, "Vacaciones")
}

func TestScheduleBlockHandler_Create_Recurring_Success(t *testing.T) {
	doJ, _ := newScheduleBlockApp(&mockScheduleRepo{})

	body := `{
		"is_recurring": true,
		"recurrence_days": [0, 6],
		"starts_at": "0001-01-01T09:00:00Z",
		"ends_at": "0001-01-01T13:00:00Z",
		"reason": "Fin de semana"
	}`
	resp := doJ("POST", "/blocks", body)
	assertStatus(t, http.StatusCreated, resp.StatusCode)
	assertBodyContains(t, resp, "recurrence_days")
}

func TestScheduleBlockHandler_Create_MissingStartsAt(t *testing.T) {
	doJ, _ := newScheduleBlockApp(&mockScheduleRepo{})

	body := `{
		"ends_at": "2026-06-01T12:00:00Z",
		"reason": "Falta starts_at"
	}`
	resp := doJ("POST", "/blocks", body)
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "starts_at")
}

func TestScheduleBlockHandler_Create_RecurringBadDay(t *testing.T) {
	doJ, _ := newScheduleBlockApp(&mockScheduleRepo{})

	body := `{
		"is_recurring": true,
		"recurrence_days": [0, 9],
		"reason": "Día inválido"
	}`
	resp := doJ("POST", "/blocks", body)
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "0 (Dom)")
}

func TestScheduleBlockHandler_List_Success(t *testing.T) {
	_, doReq := newScheduleBlockApp(&mockScheduleRepo{})

	resp := doReq("GET", "/blocks")
	assertStatus(t, http.StatusOK, resp.StatusCode)
	assertBodyContains(t, resp, "data")
}

func TestScheduleBlockHandler_Delete_Success(t *testing.T) {
	_, doReq := newScheduleBlockApp(&mockScheduleRepo{})

	resp := doReq("DELETE", "/blocks/"+testID.String())
	assertStatus(t, http.StatusNoContent, resp.StatusCode)
}
