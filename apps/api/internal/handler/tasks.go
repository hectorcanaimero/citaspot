package handler

import (
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/middleware"
)

type TaskHandler struct {
	svc      domain.TaskSvc
	validate *validator.Validate
}

func NewTaskHandler(svc domain.TaskSvc) *TaskHandler {
	return &TaskHandler{svc: svc, validate: validator.New()}
}

func (h *TaskHandler) List(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)

	q := &domain.TaskListQuery{
		Status: c.Query("status"),
	}
	if aid := c.Query("assigned_to"); aid != "" {
		id, err := uuid.Parse(aid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "assigned_to inválido"})
		}
		q.AssignedTo = &id
	}
	if cid := c.Query("customer_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "customer_id inválido"})
		}
		q.CustomerID = &id
	}
	if db := c.Query("due_before"); db != "" {
		t, err := time.Parse(time.RFC3339, db)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "due_before inválido (usar RFC3339)"})
		}
		q.DueBefore = &t
	}

	list, err := h.svc.List(c.Context(), tenantID, q)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"data": list})
}

func (h *TaskHandler) Create(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	var input domain.TaskInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Datos inválidos: " + validationMessage(err)})
	}

	t, err := h.svc.Create(c.Context(), tenantID, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.Status(http.StatusCreated).JSON(t)
}

func (h *TaskHandler) GetByID(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	t, err := h.svc.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

func (h *TaskHandler) Update(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	var input domain.TaskInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "JSON inválido"})
	}

	t, err := h.svc.Update(c.Context(), tenantID, id, &input)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

func (h *TaskHandler) Complete(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	t, err := h.svc.Complete(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}

func (h *TaskHandler) Dismiss(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "ID inválido"})
	}

	t, err := h.svc.Dismiss(c.Context(), tenantID, id)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(t)
}
