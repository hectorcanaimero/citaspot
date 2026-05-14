// Mocks para interfaces de dominio — usados en tests de handlers.
// Patrón: campo opcional fn; si es nil, retorna valor seguro por defecto.
package handler_test

import (
	"context"
	"time"

	"github.com/citaspot/api/internal/domain"
	"github.com/google/uuid"
)

// ── AuthService ───────────────────────────────────────────────────────────────

type mockAuthSvc struct {
	registerFn     func(context.Context, *domain.RegisterRequest) (*domain.RegisterResponse, error)
	loginFn        func(context.Context, *domain.LoginRequest) (*domain.LoginResponse, error)
	refreshTokenFn func(context.Context, string) (*domain.RefreshResponse, error)
}

func (m *mockAuthSvc) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.RegisterResponse, error) {
	if m.registerFn != nil {
		return m.registerFn(ctx, req)
	}
	return &domain.RegisterResponse{Token: "test-token"}, nil
}

func (m *mockAuthSvc) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, req)
	}
	return &domain.LoginResponse{Token: "test-token"}, nil
}

func (m *mockAuthSvc) RefreshToken(ctx context.Context, token string) (*domain.RefreshResponse, error) {
	if m.refreshTokenFn != nil {
		return m.refreshTokenFn(ctx, token)
	}
	return &domain.RefreshResponse{Token: "new-token"}, nil
}

// ── ProfessionalSvc ───────────────────────────────────────────────────────────

type mockProfessionalSvc struct {
	listFn        func(context.Context, uuid.UUID, bool) ([]*domain.Professional, error)
	createFn      func(context.Context, uuid.UUID, *domain.ProfessionalInput) (*domain.Professional, error)
	getByIDFn     func(context.Context, uuid.UUID, uuid.UUID) (*domain.Professional, error)
	updateFn      func(context.Context, uuid.UUID, uuid.UUID, *domain.ProfessionalInput) (*domain.Professional, error)
	getScheduleFn func(context.Context, uuid.UUID, uuid.UUID) ([]*domain.Schedule, error)
	setScheduleFn func(context.Context, uuid.UUID, uuid.UUID, []*domain.Schedule) ([]*domain.Schedule, error)
}

func (m *mockProfessionalSvc) List(ctx context.Context, tenantID uuid.UUID, includeArchived bool) ([]*domain.Professional, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, includeArchived)
	}
	return []*domain.Professional{}, nil
}

func (m *mockProfessionalSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.ProfessionalInput) (*domain.Professional, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, input)
	}
	return &domain.Professional{ID: uuid.New(), Name: input.Name}, nil
}

func (m *mockProfessionalSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Professional, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.Professional{ID: id, Name: "Test Pro"}, nil
}

func (m *mockProfessionalSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.ProfessionalInput) (*domain.Professional, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, input)
	}
	return &domain.Professional{ID: id, Name: input.Name}, nil
}

func (m *mockProfessionalSvc) GetSchedule(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Schedule, error) {
	if m.getScheduleFn != nil {
		return m.getScheduleFn(ctx, tenantID, professionalID)
	}
	return []*domain.Schedule{}, nil
}

func (m *mockProfessionalSvc) SetSchedule(ctx context.Context, tenantID, professionalID uuid.UUID, schedules []*domain.Schedule) ([]*domain.Schedule, error) {
	if m.setScheduleFn != nil {
		return m.setScheduleFn(ctx, tenantID, professionalID, schedules)
	}
	return schedules, nil
}

func (m *mockProfessionalSvc) ListServices(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Service, error) {
	return []*domain.Service{}, nil
}

func (m *mockProfessionalSvc) AssignService(ctx context.Context, tenantID, professionalID, serviceID uuid.UUID) error {
	return nil
}

func (m *mockProfessionalSvc) RemoveService(ctx context.Context, tenantID, professionalID, serviceID uuid.UUID) error {
	return nil
}

// ── ServiceSvc ────────────────────────────────────────────────────────────────

type mockServiceSvc struct {
	listFn    func(context.Context, uuid.UUID) ([]*domain.Service, error)
	createFn  func(context.Context, uuid.UUID, *domain.ServiceInput) (*domain.Service, error)
	getByIDFn func(context.Context, uuid.UUID, uuid.UUID) (*domain.Service, error)
	updateFn  func(context.Context, uuid.UUID, uuid.UUID, *domain.ServiceInput) (*domain.Service, error)
}

func (m *mockServiceSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Service, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID)
	}
	return []*domain.Service{}, nil
}

func (m *mockServiceSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.ServiceInput) (*domain.Service, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, input)
	}
	return &domain.Service{ID: uuid.New(), Name: input.Name, DurationMin: input.DurationMin}, nil
}

func (m *mockServiceSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Service, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.Service{ID: id, Name: "Test Service"}, nil
}

func (m *mockServiceSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.ServiceInput) (*domain.Service, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, input)
	}
	return &domain.Service{ID: id, Name: input.Name}, nil
}

func (m *mockServiceSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return nil
}

// ── AppointmentSvc ────────────────────────────────────────────────────────────

type mockAppointmentSvc struct {
	listFn    func(context.Context, uuid.UUID, string, string) ([]*domain.AppointmentWithDetails, error)
	getByIDFn func(context.Context, uuid.UUID, uuid.UUID) (*domain.AppointmentWithDetails, error)
	createFn  func(context.Context, uuid.UUID, *domain.CreateAppointmentRequest) (*domain.Appointment, error)
	updateFn  func(context.Context, uuid.UUID, uuid.UUID, *domain.UpdateAppointmentRequest) error
	cancelFn  func(context.Context, uuid.UUID, uuid.UUID, string) error
}

func (m *mockAppointmentSvc) List(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*domain.AppointmentWithDetails, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, date, timezone)
	}
	return []*domain.AppointmentWithDetails{}, nil
}

func (m *mockAppointmentSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AppointmentWithDetails, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.AppointmentWithDetails{Appointment: domain.Appointment{ID: id}}, nil
}

func (m *mockAppointmentSvc) Create(ctx context.Context, tenantID uuid.UUID, req *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, req)
	}
	return &domain.Appointment{ID: uuid.New()}, nil
}

func (m *mockAppointmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, req)
	}
	return nil
}

func (m *mockAppointmentSvc) Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, tenantID, id, reason)
	}
	return nil
}

func (m *mockAppointmentSvc) ListFiltered(ctx context.Context, tenantID uuid.UUID, q *domain.AppointmentListQuery) (*domain.PaginatedAppointments, error) {
	return &domain.PaginatedAppointments{Data: []*domain.AppointmentWithDetails{}}, nil
}

func (m *mockAppointmentSvc) Reschedule(ctx context.Context, tenantID, id uuid.UUID, req *domain.RescheduleRequest) error {
	return nil
}

// ── AvailabilityService ───────────────────────────────────────────────────────

type mockAvailabilitySvc struct {
	getAvailableSlotsFn func(context.Context, uuid.UUID, *domain.AvailabilityQuery) ([]*domain.TimeSlot, error)
}

func (m *mockAvailabilitySvc) GetAvailableSlots(ctx context.Context, tenantID uuid.UUID, query *domain.AvailabilityQuery) ([]*domain.TimeSlot, error) {
	if m.getAvailableSlotsFn != nil {
		return m.getAvailableSlotsFn(ctx, tenantID, query)
	}
	return []*domain.TimeSlot{}, nil
}

// ── CustomerRepository ────────────────────────────────────────────────────────

type mockCustomerRepo struct {
	findOrCreateByPhoneFn func(context.Context, uuid.UUID, string, string) (*domain.Customer, error)
	getByIDFn             func(context.Context, uuid.UUID, uuid.UUID) (*domain.Customer, error)
	listFn                func(context.Context, uuid.UUID, string, int, int) ([]*domain.Customer, error)
}

func (m *mockCustomerRepo) FindOrCreateByPhone(ctx context.Context, tenantID uuid.UUID, name, phone string) (*domain.Customer, error) {
	if m.findOrCreateByPhoneFn != nil {
		return m.findOrCreateByPhoneFn(ctx, tenantID, name, phone)
	}
	return &domain.Customer{ID: uuid.New(), Name: name}, nil
}

func (m *mockCustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.Customer{ID: id}, nil
}

func (m *mockCustomerRepo) List(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*domain.Customer, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, search, limit, offset)
	}
	return []*domain.Customer{}, nil
}

func (m *mockCustomerRepo) UpdateStage(ctx context.Context, tenantID, customerID uuid.UUID, stageID *uuid.UUID) error {
	return nil
}

func (m *mockCustomerRepo) UpdateField(ctx context.Context, tenantID, customerID uuid.UUID, field string, value any) error {
	return nil
}

// ── KnowledgeSvc ─────────────────────────────────────────────────────────────

type mockKnowledgeSvc struct {
	listFn    func(context.Context, uuid.UUID) ([]*domain.KnowledgeDocument, error)
	getByIDFn func(context.Context, uuid.UUID, uuid.UUID) (*domain.KnowledgeDocument, error)
	createFn  func(context.Context, uuid.UUID, *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error)
	updateFn  func(context.Context, uuid.UUID, uuid.UUID, *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error)
	deleteFn  func(context.Context, uuid.UUID, uuid.UUID) error
}

func (m *mockKnowledgeSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.KnowledgeDocument, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID)
	}
	return []*domain.KnowledgeDocument{}, nil
}

func (m *mockKnowledgeSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.KnowledgeDocument, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.KnowledgeDocument{ID: id, Title: "Test Doc"}, nil
}

func (m *mockKnowledgeSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, input)
	}
	return &domain.KnowledgeDocument{ID: uuid.New(), Title: input.Title, Content: input.Content}, nil
}

func (m *mockKnowledgeSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.KnowledgeDocumentInput) (*domain.KnowledgeDocument, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, input)
	}
	return &domain.KnowledgeDocument{ID: id, Title: input.Title}, nil
}

func (m *mockKnowledgeSvc) Upload(ctx context.Context, tenantID uuid.UUID, input *domain.KnowledgeUploadInput, fileName string, fileBytes []byte, fileType string) (*domain.KnowledgeDocument, error) {
	return &domain.KnowledgeDocument{ID: uuid.New(), Title: fileName}, nil
}

func (m *mockKnowledgeSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, id)
	}
	return nil
}

// ── WhatsAppSvc ───────────────────────────────────────────────────────────────

type mockWhatsAppSvc struct {
	processInboundFn        func(context.Context, string, map[string]any) error
	handleConnectionUpdateFn func(context.Context, string, string) error
}

func (m *mockWhatsAppSvc) ProcessInbound(ctx context.Context, instanceName string, payload map[string]any) error {
	if m.processInboundFn != nil {
		return m.processInboundFn(ctx, instanceName, payload)
	}
	return nil
}

func (m *mockWhatsAppSvc) HandleConnectionUpdate(ctx context.Context, instanceName, state string) error {
	if m.handleConnectionUpdateFn != nil {
		return m.handleConnectionUpdateFn(ctx, instanceName, state)
	}
	return nil
}

// ── WAClient ──────────────────────────────────────────────────────────────────

type mockWAClient struct {
	sendTextFn    func(context.Context, string, string, string) (string, error)
	isConnectedFn func(context.Context, string) (bool, error)
	connectFn     func(context.Context, string) (string, error)
	disconnectFn  func(context.Context, string) error
	setWebhookFn  func(context.Context, string, string) error
	fetchQRFn     func(context.Context, string) (string, error)
}

func (m *mockWAClient) SendText(ctx context.Context, instanceName, phone, text string) (string, error) {
	if m.sendTextFn != nil {
		return m.sendTextFn(ctx, instanceName, phone, text)
	}
	return "msg-id", nil
}

func (m *mockWAClient) IsConnected(ctx context.Context, instanceName string) (bool, error) {
	if m.isConnectedFn != nil {
		return m.isConnectedFn(ctx, instanceName)
	}
	return true, nil
}

func (m *mockWAClient) Connect(ctx context.Context, instanceName string) (string, error) {
	if m.connectFn != nil {
		return m.connectFn(ctx, instanceName)
	}
	return "", nil
}

func (m *mockWAClient) Disconnect(ctx context.Context, instanceName string) error {
	if m.disconnectFn != nil {
		return m.disconnectFn(ctx, instanceName)
	}
	return nil
}

func (m *mockWAClient) SetWebhook(ctx context.Context, instanceName, webhookURL string) error {
	if m.setWebhookFn != nil {
		return m.setWebhookFn(ctx, instanceName, webhookURL)
	}
	return nil
}

func (m *mockWAClient) FetchQR(ctx context.Context, instanceName string) (string, error) {
	if m.fetchQRFn != nil {
		return m.fetchQRFn(ctx, instanceName)
	}
	return "", nil
}

// ── AuthRepository (usado por BillingHandler) ─────────────────────────────────

type mockAuthRepo struct {
	createTenantFn          func(context.Context, *domain.Tenant) error
	createUserFn            func(context.Context, *domain.User) error
	findUserByEmailFn       func(context.Context, string) (*domain.User, *domain.Tenant, error)
	findUserByAuthIDFn      func(context.Context, string) (*domain.User, *domain.Tenant, error)
	tenantSlugExistsFn      func(context.Context, string) (bool, error)
	findTenantByIDFn        func(context.Context, uuid.UUID) (*domain.Tenant, error)
	findTenantBySlugFn      func(context.Context, string) (*domain.Tenant, error)
	updateTenantBillingFn   func(context.Context, uuid.UUID, string, string, string, string) error
	findTenantStripeIDsFn   func(context.Context, uuid.UUID) (string, string, error)
	getTenantSettingsFn     func(context.Context, uuid.UUID) (*domain.TenantSettings, error)
	updateTenantSettingsFn  func(context.Context, uuid.UUID, *domain.TenantSettings) error
	updateTenantProfileFn   func(context.Context, uuid.UUID, *domain.UpdateTenantProfileRequest) error
	updateUserProfileFn     func(context.Context, uuid.UUID, uuid.UUID, *domain.UpdateUserProfileRequest) error
}

func (m *mockAuthRepo) CreateTenant(ctx context.Context, t *domain.Tenant) error {
	if m.createTenantFn != nil {
		return m.createTenantFn(ctx, t)
	}
	return nil
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, u *domain.User) error {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, u)
	}
	return nil
}

func (m *mockAuthRepo) FindUserByEmail(ctx context.Context, email string) (*domain.User, *domain.Tenant, error) {
	if m.findUserByEmailFn != nil {
		return m.findUserByEmailFn(ctx, email)
	}
	return nil, nil, domain.ErrNotFound
}

func (m *mockAuthRepo) FindUserByAuthID(ctx context.Context, authID string) (*domain.User, *domain.Tenant, error) {
	if m.findUserByAuthIDFn != nil {
		return m.findUserByAuthIDFn(ctx, authID)
	}
	return nil, nil, domain.ErrNotFound
}

func (m *mockAuthRepo) TenantSlugExists(ctx context.Context, slug string) (bool, error) {
	if m.tenantSlugExistsFn != nil {
		return m.tenantSlugExistsFn(ctx, slug)
	}
	return false, nil
}

func (m *mockAuthRepo) FindTenantByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if m.findTenantByIDFn != nil {
		return m.findTenantByIDFn(ctx, id)
	}
	return &domain.Tenant{ID: id, Email: "owner@test.com"}, nil
}

func (m *mockAuthRepo) FindTenantBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	if m.findTenantBySlugFn != nil {
		return m.findTenantBySlugFn(ctx, slug)
	}
	return &domain.Tenant{Slug: slug}, nil
}

func (m *mockAuthRepo) UpdateTenantWAStatus(ctx context.Context, slug, status string) error {
	return nil
}

func (m *mockAuthRepo) FindConnectedTenantSlugs(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockAuthRepo) UpdateTenantBilling(ctx context.Context, tenantID uuid.UUID, plan, planStatus, stripeCustomerID, stripeSubID string) error {
	if m.updateTenantBillingFn != nil {
		return m.updateTenantBillingFn(ctx, tenantID, plan, planStatus, stripeCustomerID, stripeSubID)
	}
	return nil
}

func (m *mockAuthRepo) FindTenantStripeIDs(ctx context.Context, tenantID uuid.UUID) (string, string, error) {
	if m.findTenantStripeIDsFn != nil {
		return m.findTenantStripeIDsFn(ctx, tenantID)
	}
	return "", "", nil
}

func (m *mockAuthRepo) CompleteOnboarding(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockAuthRepo) GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSettings, error) {
	if m.getTenantSettingsFn != nil {
		return m.getTenantSettingsFn(ctx, tenantID)
	}
	return &domain.TenantSettings{}, nil
}

func (m *mockAuthRepo) UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, s *domain.TenantSettings) error {
	if m.updateTenantSettingsFn != nil {
		return m.updateTenantSettingsFn(ctx, tenantID, s)
	}
	return nil
}

func (m *mockAuthRepo) UpdateTenantProfile(ctx context.Context, tenantID uuid.UUID, req *domain.UpdateTenantProfileRequest) error {
	if m.updateTenantProfileFn != nil {
		return m.updateTenantProfileFn(ctx, tenantID, req)
	}
	return nil
}

func (m *mockAuthRepo) UpdateUserProfile(ctx context.Context, userID, tenantID uuid.UUID, req *domain.UpdateUserProfileRequest) error {
	if m.updateUserProfileFn != nil {
		return m.updateUserProfileFn(ctx, userID, tenantID, req)
	}
	return nil
}

// ── PublicSvc ─────────────────────────────────────────────────────────────────

type mockPublicSvc struct {
	getProfileFn      func(context.Context, string) (*domain.PublicProfile, error)
	getAvailabilityFn func(context.Context, string, *domain.AvailabilityQuery) ([]*domain.TimeSlot, error)
	bookFn            func(context.Context, string, *domain.CreateAppointmentRequest) (*domain.Appointment, error)
}

func (m *mockPublicSvc) GetProfile(ctx context.Context, slug string) (*domain.PublicProfile, error) {
	if m.getProfileFn != nil {
		return m.getProfileFn(ctx, slug)
	}
	return &domain.PublicProfile{Slug: slug, Name: "Test Business"}, nil
}

func (m *mockPublicSvc) GetAvailability(ctx context.Context, slug string, query *domain.AvailabilityQuery) ([]*domain.TimeSlot, error) {
	if m.getAvailabilityFn != nil {
		return m.getAvailabilityFn(ctx, slug, query)
	}
	return []*domain.TimeSlot{}, nil
}

func (m *mockPublicSvc) Book(ctx context.Context, slug string, req *domain.CreateAppointmentRequest) (*domain.Appointment, error) {
	if m.bookFn != nil {
		return m.bookFn(ctx, slug, req)
	}
	return &domain.Appointment{ID: uuid.New()}, nil
}

// ── PipelineStageSvc ─────────────────────────────────────────────────────────

type mockPipelineStageSvc struct {
	createFn  func(context.Context, uuid.UUID, *domain.PipelineStageInput) (*domain.PipelineStage, error)
	listFn    func(context.Context, uuid.UUID) ([]*domain.PipelineStage, error)
	getByIDFn func(context.Context, uuid.UUID, uuid.UUID) (*domain.PipelineStage, error)
	updateFn  func(context.Context, uuid.UUID, uuid.UUID, *domain.PipelineStageInput) (*domain.PipelineStage, error)
	deleteFn  func(context.Context, uuid.UUID, uuid.UUID) error
	reorderFn func(context.Context, uuid.UUID, []domain.ReorderStageInput) error
}

func (m *mockPipelineStageSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.PipelineStageInput) (*domain.PipelineStage, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, input)
	}
	return &domain.PipelineStage{ID: uuid.New(), Name: input.Name}, nil
}

func (m *mockPipelineStageSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.PipelineStage, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID)
	}
	return []*domain.PipelineStage{}, nil
}

func (m *mockPipelineStageSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PipelineStage, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.PipelineStage{ID: id, Name: "Test Stage"}, nil
}

func (m *mockPipelineStageSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.PipelineStageInput) (*domain.PipelineStage, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, input)
	}
	return &domain.PipelineStage{ID: id}, nil
}

func (m *mockPipelineStageSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, id)
	}
	return nil
}

func (m *mockPipelineStageSvc) Reorder(ctx context.Context, tenantID uuid.UUID, items []domain.ReorderStageInput) error {
	if m.reorderFn != nil {
		return m.reorderFn(ctx, tenantID, items)
	}
	return nil
}

// ── TreatmentSvc ─────────────────────────────────────────────────────────────

type mockTreatmentSvc struct {
	createFn       func(context.Context, uuid.UUID, *domain.TreatmentInput) (*domain.Treatment, error)
	listFn         func(context.Context, uuid.UUID, *domain.TreatmentListQuery) ([]*domain.Treatment, error)
	getByIDFn      func(context.Context, uuid.UUID, uuid.UUID) (*domain.Treatment, error)
	updateFn       func(context.Context, uuid.UUID, uuid.UUID, *domain.TreatmentInput) (*domain.Treatment, error)
	updateStatusFn func(context.Context, uuid.UUID, uuid.UUID, *domain.UpdateTreatmentStatusInput) (*domain.Treatment, error)
}

func (m *mockTreatmentSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.TreatmentInput) (*domain.Treatment, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, input)
	}
	return &domain.Treatment{ID: uuid.New(), Name: input.Name}, nil
}

func (m *mockTreatmentSvc) List(ctx context.Context, tenantID uuid.UUID, q *domain.TreatmentListQuery) ([]*domain.Treatment, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, q)
	}
	return []*domain.Treatment{}, nil
}

func (m *mockTreatmentSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Treatment, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.Treatment{ID: id, Name: "Test Treatment"}, nil
}

func (m *mockTreatmentSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.TreatmentInput) (*domain.Treatment, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, input)
	}
	return &domain.Treatment{ID: id}, nil
}

func (m *mockTreatmentSvc) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateTreatmentStatusInput) (*domain.Treatment, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, tenantID, id, input)
	}
	return &domain.Treatment{ID: id}, nil
}

// ── TreatmentSessionSvc ──────────────────────────────────────────────────────

type mockTreatmentSessionSvc struct {
	createFn  func(context.Context, uuid.UUID, uuid.UUID, *domain.TreatmentSessionInput) (*domain.TreatmentSession, error)
	listFn    func(context.Context, uuid.UUID, uuid.UUID) ([]*domain.TreatmentSession, error)
	getByIDFn func(context.Context, uuid.UUID, uuid.UUID) (*domain.TreatmentSession, error)
	updateFn  func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *domain.UpdateTreatmentSessionInput) (*domain.TreatmentSession, error)
	deleteFn  func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}

func (m *mockTreatmentSessionSvc) Create(ctx context.Context, tenantID, treatmentID uuid.UUID, input *domain.TreatmentSessionInput) (*domain.TreatmentSession, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, treatmentID, input)
	}
	return &domain.TreatmentSession{ID: uuid.New()}, nil
}

func (m *mockTreatmentSessionSvc) List(ctx context.Context, tenantID, treatmentID uuid.UUID) ([]*domain.TreatmentSession, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, treatmentID)
	}
	return []*domain.TreatmentSession{}, nil
}

func (m *mockTreatmentSessionSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TreatmentSession, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.TreatmentSession{ID: id}, nil
}

func (m *mockTreatmentSessionSvc) Update(ctx context.Context, tenantID, treatmentID, id uuid.UUID, input *domain.UpdateTreatmentSessionInput) (*domain.TreatmentSession, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, treatmentID, id, input)
	}
	return &domain.TreatmentSession{ID: id}, nil
}

func (m *mockTreatmentSessionSvc) Delete(ctx context.Context, tenantID, treatmentID, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, treatmentID, id)
	}
	return nil
}

// ── TaskSvc ──────────────────────────────────────────────────────────────────

type mockTaskSvc struct {
	createFn   func(context.Context, uuid.UUID, *domain.TaskInput) (*domain.Task, error)
	listFn     func(context.Context, uuid.UUID, *domain.TaskListQuery) ([]*domain.Task, error)
	getByIDFn  func(context.Context, uuid.UUID, uuid.UUID) (*domain.Task, error)
	updateFn   func(context.Context, uuid.UUID, uuid.UUID, *domain.TaskInput) (*domain.Task, error)
	completeFn func(context.Context, uuid.UUID, uuid.UUID) (*domain.Task, error)
	dismissFn  func(context.Context, uuid.UUID, uuid.UUID) (*domain.Task, error)
}

func (m *mockTaskSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.TaskInput) (*domain.Task, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, input)
	}
	return &domain.Task{ID: uuid.New(), Title: input.Title}, nil
}

func (m *mockTaskSvc) List(ctx context.Context, tenantID uuid.UUID, q *domain.TaskListQuery) ([]*domain.Task, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, q)
	}
	return []*domain.Task{}, nil
}

func (m *mockTaskSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.Task{ID: id, Title: "Test Task"}, nil
}

func (m *mockTaskSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.TaskInput) (*domain.Task, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, input)
	}
	return &domain.Task{ID: id}, nil
}

func (m *mockTaskSvc) Complete(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	if m.completeFn != nil {
		return m.completeFn(ctx, tenantID, id)
	}
	return &domain.Task{ID: id, Status: "completed"}, nil
}

func (m *mockTaskSvc) Dismiss(ctx context.Context, tenantID, id uuid.UUID) (*domain.Task, error) {
	if m.dismissFn != nil {
		return m.dismissFn(ctx, tenantID, id)
	}
	return &domain.Task{ID: id, Status: "dismissed"}, nil
}

// ── RuleSvc ──────────────────────────────────────────────────────────────────

type mockRuleSvc struct {
	createFn         func(context.Context, uuid.UUID, *domain.RuleInput) (*domain.Rule, error)
	listFn           func(context.Context, uuid.UUID) ([]*domain.Rule, error)
	getByIDFn        func(context.Context, uuid.UUID, uuid.UUID) (*domain.Rule, error)
	updateFn         func(context.Context, uuid.UUID, uuid.UUID, *domain.RuleInput) (*domain.Rule, error)
	deleteFn         func(context.Context, uuid.UUID, uuid.UUID) error
	listExecutionsFn func(context.Context, uuid.UUID, uuid.UUID, int) ([]*domain.RuleExecution, error)
}

func (m *mockRuleSvc) Create(ctx context.Context, tenantID uuid.UUID, input *domain.RuleInput) (*domain.Rule, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, input)
	}
	return &domain.Rule{ID: uuid.New(), Name: input.Name}, nil
}

func (m *mockRuleSvc) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Rule, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID)
	}
	return []*domain.Rule{}, nil
}

func (m *mockRuleSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Rule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.Rule{ID: id, Name: "Test Rule"}, nil
}

func (m *mockRuleSvc) Update(ctx context.Context, tenantID, id uuid.UUID, input *domain.RuleInput) (*domain.Rule, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, input)
	}
	return &domain.Rule{ID: id}, nil
}

func (m *mockRuleSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, id)
	}
	return nil
}

func (m *mockRuleSvc) ListExecutions(ctx context.Context, tenantID, ruleID uuid.UUID, limit int) ([]*domain.RuleExecution, error) {
	if m.listExecutionsFn != nil {
		return m.listExecutionsFn(ctx, tenantID, ruleID, limit)
	}
	return []*domain.RuleExecution{}, nil
}

// ── ScheduleRepository ───────────────────────────────────────────────────────

type mockScheduleRepo struct {
	getSchedulesFn    func(context.Context, uuid.UUID, uuid.UUID) ([]*domain.Schedule, error)
	upsertSchedulesFn func(context.Context, uuid.UUID, uuid.UUID, []*domain.Schedule) ([]*domain.Schedule, error)
	getBlocksFn       func(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time) ([]*domain.ScheduleBlock, error)
	getApptsFn        func(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time) ([]*domain.Appointment, error)
	createBlockFn     func(context.Context, *domain.ScheduleBlock) error
	listBlocksFn      func(context.Context, uuid.UUID, *uuid.UUID) ([]*domain.ScheduleBlock, error)
	deleteBlockFn     func(context.Context, uuid.UUID, uuid.UUID) error
}

func (m *mockScheduleRepo) GetSchedules(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Schedule, error) {
	if m.getSchedulesFn != nil {
		return m.getSchedulesFn(ctx, tenantID, professionalID)
	}
	return []*domain.Schedule{}, nil
}

func (m *mockScheduleRepo) UpsertSchedules(ctx context.Context, tenantID, professionalID uuid.UUID, schedules []*domain.Schedule) ([]*domain.Schedule, error) {
	if m.upsertSchedulesFn != nil {
		return m.upsertSchedulesFn(ctx, tenantID, professionalID, schedules)
	}
	return schedules, nil
}

func (m *mockScheduleRepo) GetBlocks(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*domain.ScheduleBlock, error) {
	if m.getBlocksFn != nil {
		return m.getBlocksFn(ctx, tenantID, professionalID, from, to)
	}
	return []*domain.ScheduleBlock{}, nil
}

func (m *mockScheduleRepo) GetAppointmentsInRange(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*domain.Appointment, error) {
	if m.getApptsFn != nil {
		return m.getApptsFn(ctx, tenantID, professionalID, from, to)
	}
	return []*domain.Appointment{}, nil
}

func (m *mockScheduleRepo) CreateBlock(ctx context.Context, b *domain.ScheduleBlock) error {
	if m.createBlockFn != nil {
		return m.createBlockFn(ctx, b)
	}
	return nil
}

func (m *mockScheduleRepo) ListBlocks(ctx context.Context, tenantID uuid.UUID, professionalID *uuid.UUID) ([]*domain.ScheduleBlock, error) {
	if m.listBlocksFn != nil {
		return m.listBlocksFn(ctx, tenantID, professionalID)
	}
	return []*domain.ScheduleBlock{}, nil
}

func (m *mockScheduleRepo) DeleteBlock(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteBlockFn != nil {
		return m.deleteBlockFn(ctx, tenantID, id)
	}
	return nil
}

// ── CRMMetricsRepository ────────────────────────────────────────────────────

type mockCRMMetricsRepo struct {
	getMetricsFn func(context.Context, uuid.UUID) (*domain.CRMMetrics, error)
}

func (m *mockCRMMetricsRepo) GetMetrics(ctx context.Context, tenantID uuid.UUID) (*domain.CRMMetrics, error) {
	if m.getMetricsFn != nil {
		return m.getMetricsFn(ctx, tenantID)
	}
	return &domain.CRMMetrics{}, nil
}

// ── ClinicalNoteSvc ─────────────────────────────────────────────────────────

type mockClinicalNoteSvc struct {
	createFn           func(context.Context, uuid.UUID, uuid.UUID, *domain.CreateClinicalNoteRequest) (*domain.ClinicalNote, error)
	getByIDFn          func(context.Context, uuid.UUID, uuid.UUID) (*domain.ClinicalNoteWithDetails, error)
	getByAppointmentFn func(context.Context, uuid.UUID, uuid.UUID) (*domain.ClinicalNoteWithDetails, error)
	listByCustomerFn   func(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*domain.ClinicalNoteWithDetails, int, error)
	updateFn           func(context.Context, uuid.UUID, uuid.UUID, *domain.UpdateClinicalNoteRequest) error
	deleteFn           func(context.Context, uuid.UUID, uuid.UUID) error
}

func (m *mockClinicalNoteSvc) Create(ctx context.Context, tenantID, appointmentID uuid.UUID, req *domain.CreateClinicalNoteRequest) (*domain.ClinicalNote, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, appointmentID, req)
	}
	return &domain.ClinicalNote{ID: uuid.New(), AppointmentID: appointmentID}, nil
}

func (m *mockClinicalNoteSvc) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, id)
	}
	return &domain.ClinicalNoteWithDetails{ClinicalNote: domain.ClinicalNote{ID: id}}, nil
}

func (m *mockClinicalNoteSvc) GetByAppointmentID(ctx context.Context, tenantID, appointmentID uuid.UUID) (*domain.ClinicalNoteWithDetails, error) {
	if m.getByAppointmentFn != nil {
		return m.getByAppointmentFn(ctx, tenantID, appointmentID)
	}
	return &domain.ClinicalNoteWithDetails{ClinicalNote: domain.ClinicalNote{ID: uuid.New(), AppointmentID: appointmentID}}, nil
}

func (m *mockClinicalNoteSvc) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit, offset int) ([]*domain.ClinicalNoteWithDetails, int, error) {
	if m.listByCustomerFn != nil {
		return m.listByCustomerFn(ctx, tenantID, customerID, limit, offset)
	}
	return []*domain.ClinicalNoteWithDetails{}, 0, nil
}

func (m *mockClinicalNoteSvc) Update(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateClinicalNoteRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, id, req)
	}
	return nil
}

func (m *mockClinicalNoteSvc) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, id)
	}
	return nil
}

// ── ClinicalFileSvc ─────────────────────────────────────────────────────────

type mockClinicalFileSvc struct {
	uploadFn         func(context.Context, uuid.UUID, uuid.UUID, domain.ClinicalFileUploadInput) (*domain.ClinicalFile, error)
	listByNoteFn     func(context.Context, uuid.UUID, uuid.UUID) ([]*domain.ClinicalFile, error)
	listByCustomerFn func(context.Context, uuid.UUID, uuid.UUID, string, int, int) ([]*domain.ClinicalFile, int, error)
	deleteFn         func(context.Context, uuid.UUID, uuid.UUID) error
}

func (m *mockClinicalFileSvc) Upload(ctx context.Context, tenantID, noteID uuid.UUID, input domain.ClinicalFileUploadInput) (*domain.ClinicalFile, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, tenantID, noteID, input)
	}
	return &domain.ClinicalFile{ID: uuid.New(), ClinicalNoteID: noteID}, nil
}

func (m *mockClinicalFileSvc) ListByNote(ctx context.Context, tenantID, noteID uuid.UUID) ([]*domain.ClinicalFile, error) {
	if m.listByNoteFn != nil {
		return m.listByNoteFn(ctx, tenantID, noteID)
	}
	return []*domain.ClinicalFile{}, nil
}

func (m *mockClinicalFileSvc) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, category string, limit, offset int) ([]*domain.ClinicalFile, int, error) {
	if m.listByCustomerFn != nil {
		return m.listByCustomerFn(ctx, tenantID, customerID, category, limit, offset)
	}
	return []*domain.ClinicalFile{}, 0, nil
}

func (m *mockClinicalFileSvc) Delete(ctx context.Context, tenantID, fileID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, fileID)
	}
	return nil
}

// ── BrandingService ─────────────────────────────────────────────────────────

type mockBrandingSvc struct {
	uploadAssetFn func(context.Context, uuid.UUID, *domain.BrandingUploadInput) (string, error)
	removeAssetFn func(context.Context, uuid.UUID, domain.BrandingAssetKind) error
}

func (m *mockBrandingSvc) UploadAsset(ctx context.Context, tenantID uuid.UUID, in *domain.BrandingUploadInput) (string, error) {
	if m.uploadAssetFn != nil {
		return m.uploadAssetFn(ctx, tenantID, in)
	}
	return "https://cdn.example.com/branding/logo.png", nil
}

func (m *mockBrandingSvc) RemoveAsset(ctx context.Context, tenantID uuid.UUID, kind domain.BrandingAssetKind) error {
	if m.removeAssetFn != nil {
		return m.removeAssetFn(ctx, tenantID, kind)
	}
	return nil
}
