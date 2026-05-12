package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newSettingsApp(repo *mockAuthRepo) (*handler.SettingsHandler, func(string, string, string) *http.Response, func(string, string) *http.Response) {
	h := handler.NewSettingsHandler(repo)
	app := newProtectedApp()
	app.Get("/settings", h.Get)
	app.Put("/settings", h.Update)
	app.Put("/settings/business", h.UpdateTenantProfile)
	app.Put("/settings/profile", h.UpdateMyProfile)
	return h, func(method, path, body string) *http.Response {
			return doJSON(app, method, path, body)
		}, func(method, path string) *http.Response {
			return doRequest(app, method, path)
		}
}

func TestSettingsHandler_Get_Success(t *testing.T) {
	repo := &mockAuthRepo{}
	_, _, doReq := newSettingsApp(repo)

	resp := doReq("GET", "/settings")
	assertStatus(t, http.StatusOK, resp.StatusCode)
}

func TestSettingsHandler_Update_Success(t *testing.T) {
	called := false
	repo := &mockAuthRepo{}
	repo.updateTenantSettingsFn = func(_ context.Context, _ uuid.UUID, _ *domain.TenantSettings) error {
		called = true
		return nil
	}
	_, doJ, _ := newSettingsApp(repo)

	resp := doJ("PUT", "/settings", `{"reminder_minutes":[30,60],"bot_name":"TestBot"}`)
	assertStatus(t, http.StatusNoContent, resp.StatusCode)
	if !called {
		t.Error("expected UpdateTenantSettings to be called")
	}
}

func TestSettingsHandler_Update_BadJSON(t *testing.T) {
	_, doJ, _ := newSettingsApp(&mockAuthRepo{})

	resp := doJ("PUT", "/settings", `{bad}`)
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "formato")
}

func TestSettingsHandler_Update_TooManyReminders(t *testing.T) {
	_, doJ, _ := newSettingsApp(&mockAuthRepo{})

	resp := doJ("PUT", "/settings", `{"reminder_minutes":[15,30,60,120,240,480]}`)
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "máximo 5")
}

func TestSettingsHandler_Update_ReminderTooSmall(t *testing.T) {
	_, doJ, _ := newSettingsApp(&mockAuthRepo{})

	resp := doJ("PUT", "/settings", `{"reminder_minutes":[5]}`)
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "15 minutos")
}

func TestSettingsHandler_UpdateTenantProfile_Success(t *testing.T) {
	called := false
	repo := &mockAuthRepo{}
	repo.updateTenantProfileFn = func(_ context.Context, _ uuid.UUID, _ *domain.UpdateTenantProfileRequest) error {
		called = true
		return nil
	}
	_, doJ, _ := newSettingsApp(repo)

	resp := doJ("PUT", "/settings/business", `{"name":"New Biz Name","phone":"+18091234567"}`)
	assertStatus(t, http.StatusNoContent, resp.StatusCode)
	if !called {
		t.Error("expected UpdateTenantProfile to be called")
	}
}

func TestSettingsHandler_UpdateMyProfile_Success(t *testing.T) {
	called := false
	repo := &mockAuthRepo{}
	repo.updateUserProfileFn = func(_ context.Context, _, _ uuid.UUID, _ *domain.UpdateUserProfileRequest) error {
		called = true
		return nil
	}
	_, doJ, _ := newSettingsApp(repo)

	resp := doJ("PUT", "/settings/profile", `{"name":"New Name"}`)
	assertStatus(t, http.StatusNoContent, resp.StatusCode)
	if !called {
		t.Error("expected UpdateUserProfile to be called")
	}
}

func TestSettingsHandler_UpdateMyProfile_BadJSON(t *testing.T) {
	_, doJ, _ := newSettingsApp(&mockAuthRepo{})

	resp := doJ("PUT", "/settings/profile", `{bad}`)
	assertStatus(t, http.StatusBadRequest, resp.StatusCode)
	assertBodyContains(t, resp, "formato")
}
