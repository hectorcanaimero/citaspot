package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/google/uuid"
)

func newKnowledgeHandler(svc *mockKnowledgeSvc) *handler.KnowledgeHandler {
	return handler.NewKnowledgeHandler(svc)
}

const validKnowledgeBody = `{
	"category":"faq",
	"title":"Horarios de atención",
	"content":"Atendemos de lunes a viernes de 9am a 6pm y sábados de 9am a 2pm."
}`

func TestKnowledgeHandler_List(t *testing.T) {
	t.Run("success empty list", func(t *testing.T) {
		h := newKnowledgeHandler(&mockKnowledgeSvc{})
		app := newProtectedApp()
		app.Get("/knowledge", h.List)

		resp := doRequest(app, "GET", "/knowledge")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "data")
	})

	t.Run("success with documents", func(t *testing.T) {
		h := newKnowledgeHandler(&mockKnowledgeSvc{
			listFn: func(_ context.Context, _ uuid.UUID) ([]*domain.KnowledgeDocument, error) {
				return []*domain.KnowledgeDocument{
					{ID: testID, Title: "FAQ General", Category: "faq"},
				}, nil
			},
		})
		app := newProtectedApp()
		app.Get("/knowledge", h.List)

		resp := doRequest(app, "GET", "/knowledge")
		assertStatus(t, http.StatusOK, resp.StatusCode)
		assertBodyContains(t, resp, "FAQ General")
	})
}

func TestKnowledgeHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.KnowledgeDocument, error)
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
			mockFn: func(_ context.Context, _, _ uuid.UUID) (*domain.KnowledgeDocument, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newKnowledgeHandler(&mockKnowledgeSvc{getByIDFn: tc.mockFn})
			app := newProtectedApp()
			app.Get("/knowledge/:id", h.Get)

			resp := doRequest(app, "GET", "/knowledge/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestKnowledgeHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockFn     func(context.Context, uuid.UUID, *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			body:       validKnowledgeBody,
			wantStatus: http.StatusCreated,
			wantBody:   "Horarios de atención",
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "JSON",
		},
		{
			name:       "invalid category",
			body:       `{"category":"invalid","title":"Test","content":"This is a content longer than 10 chars"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing title",
			body:       `{"category":"faq","content":"This is a content longer than 10 chars"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "content too short",
			body:       `{"category":"faq","title":"Test Title","content":"Short"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "service error",
			body: validKnowledgeBody,
			mockFn: func(_ context.Context, _ uuid.UUID, _ *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error) {
				return nil, domain.ErrInternal
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newKnowledgeHandler(&mockKnowledgeSvc{createFn: tc.mockFn})
			app := newProtectedApp()
			app.Post("/knowledge", h.Create)

			resp := doJSON(app, "POST", "/knowledge", tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestKnowledgeHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			body:       validKnowledgeBody,
			wantStatus: http.StatusOK,
			wantBody:   "Horarios de atención",
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			body:       validKnowledgeBody,
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
			name:       "invalid category",
			id:         testID.String(),
			body:       `{"category":"invalid","title":"Test","content":"This is valid content length"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "not found",
			id:   testID.String(),
			body: validKnowledgeBody,
			mockFn: func(_ context.Context, _, _ uuid.UUID, _ *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error) {
				return nil, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newKnowledgeHandler(&mockKnowledgeSvc{updateFn: tc.mockFn})
			app := newProtectedApp()
			app.Put("/knowledge/:id", h.Update)

			resp := doJSON(app, "PUT", "/knowledge/"+tc.id, tc.body)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}

func TestKnowledgeHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(context.Context, uuid.UUID, uuid.UUID) error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			id:         testID.String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			id:         "bad-uuid",
			wantStatus: http.StatusBadRequest,
			wantBody:   "ID",
		},
		{
			name: "not found",
			id:   testID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) error {
				return domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newKnowledgeHandler(&mockKnowledgeSvc{deleteFn: tc.mockFn})
			app := newProtectedApp()
			app.Delete("/knowledge/:id", h.Delete)

			resp := doRequest(app, "DELETE", "/knowledge/"+tc.id)
			assertStatus(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assertBodyContains(t, resp, tc.wantBody)
			}
		})
	}
}
