// Mocks para interfaces de dominio — usados en tests de handlers.
// Patrón: campo opcional fn; si es nil, retorna valor seguro por defecto.
package handler_test

import (
	"context"

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

func (m *mockCustomerRepo) UpdateStage(ctx context.Context, tenantID, customerID, stageID uuid.UUID) error {
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
	connectFn     func(context.Context, string) error
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

func (m *mockWAClient) Connect(ctx context.Context, instanceName string) error {
	if m.connectFn != nil {
		return m.connectFn(ctx, instanceName)
	}
	return nil
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
	return &domain.TenantSettings{}, nil
}

func (m *mockAuthRepo) UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, s *domain.TenantSettings) error {
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
