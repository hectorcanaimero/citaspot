// Interfaces de repositorios y servicios — facilitan el testing con mocks.
package domain

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// ── Auth ──────────────────────────────────────────────────────────────────────

// AuthRepository operaciones DB para autenticación.
type AuthRepository interface {
	CreateTenant(ctx context.Context, t *Tenant) error
	CreateUser(ctx context.Context, u *User) error
	FindUserByEmail(ctx context.Context, email string) (*User, *Tenant, error)
	FindUserByAuthID(ctx context.Context, authID string) (*User, *Tenant, error)
	TenantSlugExists(ctx context.Context, slug string) (bool, error)
	FindTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	FindTenantBySlug(ctx context.Context, slug string) (*Tenant, error)
	// CompleteOnboarding marca el onboarding del tenant como completado.
	CompleteOnboarding(ctx context.Context, tenantID uuid.UUID) error
	// UpdateTenantWAStatus actualiza el wa_status del tenant por slug.
	// Valores válidos: "connected", "disconnected", "banned".
	UpdateTenantWAStatus(ctx context.Context, slug, status string) error
	// FindConnectedTenantSlugs retorna los slugs de tenants con wa_status = 'connected'.
	// Usado al startup para re-registrar webhooks en Evolution API.
	FindConnectedTenantSlugs(ctx context.Context) ([]string, error)
	// UpdateTenantBilling actualiza plan, plan_status y datos de Stripe tras un evento de pago.
	// stripeCustomerID y stripeSubID pueden estar vacíos si aún no se completó el checkout.
	UpdateTenantBilling(ctx context.Context, tenantID uuid.UUID, plan, planStatus, stripeCustomerID, stripeSubID string) error
	// FindTenantStripeIDs retorna los IDs de Stripe (customer, subscription) del tenant.
	// Los valores pueden estar vacíos si el tenant aún no tiene suscripción activa.
	FindTenantStripeIDs(ctx context.Context, tenantID uuid.UUID) (customerID, subID string, err error)
	GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (*TenantSettings, error)
	UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, s *TenantSettings) error
}

// AuthService lógica de negocio de autenticación.
type AuthService interface {
	Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshResponse, error)
}

// ── Professionals ─────────────────────────────────────────────────────────────

// ProfessionalRepository operaciones DB para profesionales.
type ProfessionalRepository interface {
	List(ctx context.Context, tenantID uuid.UUID, includeArchived bool) ([]*Professional, error)
	Create(ctx context.Context, p *Professional) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Professional, error)
	Update(ctx context.Context, p *Professional) error
	ListByTenantPublic(ctx context.Context, tenantID uuid.UUID) ([]*Professional, error)
}

// ProfessionalSvc lógica de negocio para profesionales.
type ProfessionalSvc interface {
	List(ctx context.Context, tenantID uuid.UUID, includeArchived bool) ([]*Professional, error)
	Create(ctx context.Context, tenantID uuid.UUID, input *ProfessionalInput) (*Professional, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Professional, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *ProfessionalInput) (*Professional, error)
	GetSchedule(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*Schedule, error)
	SetSchedule(ctx context.Context, tenantID, professionalID uuid.UUID, schedules []*Schedule) ([]*Schedule, error)
}

// ── Services ──────────────────────────────────────────────────────────────────

// ServiceRepository operaciones DB para servicios.
type ServiceRepository interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*Service, error)
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*Service, error)
	Create(ctx context.Context, s *Service) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Service, error)
	Update(ctx context.Context, s *Service) error
}

// ServiceSvc lógica de negocio para servicios.
type ServiceSvc interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*Service, error)
	Create(ctx context.Context, tenantID uuid.UUID, input *ServiceInput) (*Service, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Service, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *ServiceInput) (*Service, error)
}

// ── Schedules / Availability ──────────────────────────────────────────────────

// ScheduleRepository operaciones DB para horarios y disponibilidad.
type ScheduleRepository interface {
	GetSchedules(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*Schedule, error)
	UpsertSchedules(ctx context.Context, tenantID, professionalID uuid.UUID, schedules []*Schedule) ([]*Schedule, error)
	GetBlocks(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*ScheduleBlock, error)
	GetAppointmentsInRange(ctx context.Context, tenantID, professionalID uuid.UUID, from, to time.Time) ([]*Appointment, error)
	CreateBlock(ctx context.Context, b *ScheduleBlock) error
	ListBlocks(ctx context.Context, tenantID uuid.UUID, professionalID *uuid.UUID) ([]*ScheduleBlock, error)
	DeleteBlock(ctx context.Context, tenantID, id uuid.UUID) error
}

// ScheduleBlockRepository operaciones CRUD para bloqueos de horario.
type ScheduleBlockRepository interface {
	Create(ctx context.Context, b *ScheduleBlock) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, professionalID *uuid.UUID) ([]*ScheduleBlock, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// AvailabilityService calcula slots disponibles.
type AvailabilityService interface {
	GetAvailableSlots(ctx context.Context, tenantID uuid.UUID, query *AvailabilityQuery) ([]*TimeSlot, error)
}

// ── Customers ─────────────────────────────────────────────────────────────────

// CustomerRepository operaciones DB para clientes.
type CustomerRepository interface {
	FindOrCreateByPhone(ctx context.Context, tenantID uuid.UUID, name, phone string) (*Customer, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error)
	List(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*Customer, error)
	UpdateStage(ctx context.Context, tenantID, customerID, stageID uuid.UUID) error
	UpdateField(ctx context.Context, tenantID, customerID uuid.UUID, field string, value any) error
}

// ── Appointments ──────────────────────────────────────────────────────────────

// AppointmentRepository operaciones DB para citas.
type AppointmentRepository interface {
	Create(ctx context.Context, a *Appointment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*AppointmentWithDetails, error)
	ListByDate(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*AppointmentWithDetails, error)
	ListFiltered(ctx context.Context, tenantID uuid.UUID, q *AppointmentListQuery) (*PaginatedAppointments, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, req *UpdateAppointmentRequest) error
	CheckConflict(ctx context.Context, tenantID, professionalID uuid.UUID, startsAt, endsAt time.Time, excludeID *uuid.UUID) (bool, error)
	Reschedule(ctx context.Context, tenantID, id, professionalID uuid.UUID, startsAt, endsAt time.Time) error
}

// AppointmentSvc lógica de negocio para citas.
type AppointmentSvc interface {
	List(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*AppointmentWithDetails, error)
	ListFiltered(ctx context.Context, tenantID uuid.UUID, q *AppointmentListQuery) (*PaginatedAppointments, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*AppointmentWithDetails, error)
	Create(ctx context.Context, tenantID uuid.UUID, req *CreateAppointmentRequest) (*Appointment, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, req *UpdateAppointmentRequest) error
	Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error
	Reschedule(ctx context.Context, tenantID, id uuid.UUID, req *RescheduleRequest) error
}

// ── Public (sin auth) ─────────────────────────────────────────────────────────

// PublicSvc operaciones de booking sin autenticación.
type PublicSvc interface {
	GetProfile(ctx context.Context, slug string) (*PublicProfile, error)
	GetAvailability(ctx context.Context, slug string, query *AvailabilityQuery) ([]*TimeSlot, error)
	Book(ctx context.Context, slug string, req *CreateAppointmentRequest) (*Appointment, error)
}

// ── WhatsApp / Conversaciones ─────────────────────────────────────────────────

// ConversationRepository operaciones DB para conversaciones y mensajes.
type ConversationRepository interface {
	FindOrCreateConversation(ctx context.Context, tenantID uuid.UUID, waPhone string) (*Conversation, error)
	SaveMessage(ctx context.Context, m *Message) error
	GetRecentMessages(ctx context.Context, tenantID, conversationID uuid.UUID, limit int) ([]*Message, error)
	UpdateConversationTimestamp(ctx context.Context, conversationID uuid.UUID) error
}

// NotificationRepository operaciones DB para logs de notificaciones.
type NotificationRepository interface {
	LogNotification(ctx context.Context, log *NotificationLog) error
	MarkReminderSent(ctx context.Context, appointmentID uuid.UUID, minutesBefore int) error
}

// ReminderRepository queries para encontrar citas que necesitan recordatorio.
// Nota: opera sin RLS (acceso global, no por tenant).
type ReminderRepository interface {
	FindDueReminders(ctx context.Context, minutesBefore int) ([]*ReminderJob, error)
	GetDistinctReminderMinutes(ctx context.Context) ([]int, error)
}

// MessagePublisher publica mensajes en RabbitMQ.
type MessagePublisher interface {
	Publish(ctx context.Context, queue string, body []byte) error
	Close() error
}

// WAClient envía mensajes vía Evolution API.
type WAClient interface {
	SendText(ctx context.Context, instanceName, phone, text string) (string, error)
	IsConnected(ctx context.Context, instanceName string) (bool, error)
	// Connect crea la instancia si no existe, configura el webhook y conecta.
	Connect(ctx context.Context, instanceName string) error
	// Disconnect cierra la sesión de WhatsApp del tenant.
	Disconnect(ctx context.Context, instanceName string) error
	// SetWebhook configura la URL de webhook para una instancia en Evolution.
	SetWebhook(ctx context.Context, instanceName, webhookURL string) error
	// FetchQR obtiene el QR code actual de la instancia directamente de Evolution API.
	FetchQR(ctx context.Context, instanceName string) (string, error)
}

// WhatsAppSvc procesa mensajes entrantes del webhook.
type WhatsAppSvc interface {
	ProcessInbound(ctx context.Context, instanceName string, payload map[string]any) error
	// HandleConnectionUpdate persiste el estado de conexión WA en la DB.
	// state "open" → "connected"; "close" → "disconnected".
	HandleConnectionUpdate(ctx context.Context, instanceName, state string) error
}

// ── Knowledge Base ─────────────────────────────────────────────────────────────

// KnowledgeRepository operaciones DB para la base de conocimiento.
type KnowledgeRepository interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*KnowledgeDocument, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*KnowledgeDocument, error)
	Create(ctx context.Context, doc *KnowledgeDocument) error
	Update(ctx context.Context, doc *KnowledgeDocument) error
	// UpdateContent actualiza el contenido extraído y el status tras parsear un archivo.
	UpdateContent(ctx context.Context, tenantID, id uuid.UUID, content, status string) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// KnowledgeSvc lógica de negocio para la base de conocimiento.
type KnowledgeSvc interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*KnowledgeDocument, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*KnowledgeDocument, error)
	Create(ctx context.Context, tenantID uuid.UUID, input *KnowledgeDocumentInput) (*KnowledgeDocument, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *KnowledgeDocumentInput) (*KnowledgeDocument, error)
	// Upload crea un documento desde un archivo (PDF/DOCX/XLSX) y dispara la extracción async.
	Upload(ctx context.Context, tenantID uuid.UUID, input *KnowledgeUploadInput, fileName string, fileBytes []byte, fileType string) (*KnowledgeDocument, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// ── Pipeline Stages ──────────────────────────────────────────────────────────

// PipelineStageRepository operaciones DB para etapas del pipeline.
type PipelineStageRepository interface {
	Create(ctx context.Context, s *PipelineStage) error
	List(ctx context.Context, tenantID uuid.UUID) ([]*PipelineStage, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*PipelineStage, error)
	Update(ctx context.Context, s *PipelineStage) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Reorder(ctx context.Context, tenantID uuid.UUID, items []ReorderStageInput) error
}

// PipelineStageSvc logica de negocio para etapas del pipeline.
type PipelineStageSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *PipelineStageInput) (*PipelineStage, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*PipelineStage, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*PipelineStage, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *PipelineStageInput) (*PipelineStage, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Reorder(ctx context.Context, tenantID uuid.UUID, items []ReorderStageInput) error
}

// ── Treatments ───────────────────────────────────────────────────────────────

// TreatmentRepository operaciones DB para tratamientos.
type TreatmentRepository interface {
	Create(ctx context.Context, t *Treatment) error
	List(ctx context.Context, tenantID uuid.UUID, q *TreatmentListQuery) ([]*Treatment, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Treatment, error)
	Update(ctx context.Context, t *Treatment) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
}

// TreatmentSvc logica de negocio para tratamientos.
type TreatmentSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *TreatmentInput) (*Treatment, error)
	List(ctx context.Context, tenantID uuid.UUID, q *TreatmentListQuery) ([]*Treatment, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Treatment, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *TreatmentInput) (*Treatment, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, input *UpdateTreatmentStatusInput) (*Treatment, error)
}

// ── Tasks ────────────────────────────────────────────────────────────────────

// TaskRepository operaciones DB para tareas.
type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	List(ctx context.Context, tenantID uuid.UUID, q *TaskListQuery) ([]*Task, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
	Update(ctx context.Context, t *Task) error
	Complete(ctx context.Context, tenantID, id uuid.UUID) error
	Dismiss(ctx context.Context, tenantID, id uuid.UUID) error
}

// TaskSvc logica de negocio para tareas.
type TaskSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *TaskInput) (*Task, error)
	List(ctx context.Context, tenantID uuid.UUID, q *TaskListQuery) ([]*Task, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *TaskInput) (*Task, error)
	Complete(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
	Dismiss(ctx context.Context, tenantID, id uuid.UUID) (*Task, error)
}

// ── Rules ────────────────────────────────────────────────────────────────────

// RuleRepository operaciones DB para reglas de automatizacion.
type RuleRepository interface {
	Create(ctx context.Context, r *Rule) error
	List(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Rule, error)
	Update(ctx context.Context, r *Rule) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	ListTemplates(ctx context.Context) ([]*Rule, error)
	ListActiveByTriggerEvent(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*Rule, error)
	ListActiveTemporal(ctx context.Context) ([]*Rule, error)
}

// RuleSvc logica de negocio para reglas de automatizacion (CRUD only en Phase 6A).
type RuleSvc interface {
	Create(ctx context.Context, tenantID uuid.UUID, input *RuleInput) (*Rule, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*Rule, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Rule, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, input *RuleInput) (*Rule, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ListExecutions(ctx context.Context, tenantID, ruleID uuid.UUID, limit int) ([]*RuleExecution, error)
}

// RuleExecutionRepository operaciones DB para logs de ejecucion de reglas.
type RuleExecutionRepository interface {
	Create(ctx context.Context, e *RuleExecution) error
	ListByRule(ctx context.Context, tenantID, ruleID uuid.UUID, limit int) ([]*RuleExecution, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, limit int) ([]*RuleExecution, error)
	HasRecentExecution(ctx context.Context, tenantID, ruleID, customerID uuid.UUID, cooldownHours int) (bool, error)
}

// ── CRM Metrics ─────────────────────────────────────────────────────────────

// CRMMetricsRepository operaciones DB para metricas agregadas del CRM.
type CRMMetricsRepository interface {
	GetMetrics(ctx context.Context, tenantID uuid.UUID) (*CRMMetrics, error)
}

// ── Branding ─────────────────────────────────────────────────────────────────

// BrandingAssetKind identifica el tipo de asset de branding del tenant.
type BrandingAssetKind string

const (
	BrandingAssetLogo  BrandingAssetKind = "logo"
	BrandingAssetCover BrandingAssetKind = "cover"
)

// BrandingUploadInput representa la subida de un asset de branding.
type BrandingUploadInput struct {
	Kind        BrandingAssetKind
	Filename    string
	ContentType string
	Reader      io.Reader
	Size        int64
}

// BrandingService gestiona logo, portada y descripción del tenant para la página pública.
type BrandingService interface {
	UploadAsset(ctx context.Context, tenantID uuid.UUID, in *BrandingUploadInput) (url string, err error)
	RemoveAsset(ctx context.Context, tenantID uuid.UUID, kind BrandingAssetKind) error
}
