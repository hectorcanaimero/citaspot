// Package domain contiene los tipos, interfaces y errores de dominio del Core API.
// Sin dependencias externas — solo stdlib y uuid.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ── Tenant / User ─────────────────────────────────────────────────────────────

// Tenant representa un negocio cliente de CitaSpot.
type Tenant struct {
	ID           uuid.UUID  `json:"id"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	BusinessType string     `json:"business_type"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone,omitempty"`
	City         string     `json:"city,omitempty"`
	Country      string     `json:"country"`
	Timezone     string     `json:"timezone"`
	Plan           string     `json:"plan"`
	PlanStatus     string     `json:"plan_status"`
	OnboardingDone bool       `json:"onboarding_done"`
	TrialEndsAt    *time.Time `json:"trial_ends_at,omitempty"`
	WAStatus       string         `json:"wa_status,omitempty"`
	Settings       TenantSettings `json:"settings"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TenantSettings configuración personalizable del tenant (almacenada en JSONB).
type TenantSettings struct {
	ReminderMinutes    []int  `json:"reminder_minutes"`
	BookingIntroText   string `json:"booking_intro_text"`
	BookingSuccessText string `json:"booking_success_text"`
	BotName            string `json:"bot_name"`
	BotGreeting        string `json:"bot_greeting"`
	// Branding de la página pública de reservas
	LogoURL     string `json:"logo_url"`
	CoverURL    string `json:"cover_url"`
	Description string `json:"description"`
}

// User representa un usuario con acceso al dashboard.
type User struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	AuthID    string    `json:"-"` // Supabase Auth UID — no exponer
	CreatedAt time.Time `json:"created_at"`
}

// UserDTO representación pública de un usuario.
type UserDTO struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
}

// TenantDTO representación pública de un tenant.
type TenantDTO struct {
	ID             uuid.UUID  `json:"id"`
	Slug           string     `json:"slug"`
	Name           string     `json:"name"`
	BusinessType   string     `json:"business_type"`
	Plan           string     `json:"plan"`
	PlanStatus     string     `json:"plan_status"`
	OnboardingDone bool       `json:"onboarding_done"`
	TrialEndsAt    *time.Time `json:"trial_ends_at,omitempty"`
}

// ── Auth requests / responses ─────────────────────────────────────────────────

// RegisterRequest crea un nuevo tenant y su usuario propietario.
type RegisterRequest struct {
	Email        string `json:"email"         validate:"required,email"`
	Password     string `json:"password"      validate:"required,min=8"`
	Name         string `json:"name"          validate:"required,min=2"`
	BusinessName string `json:"business_name" validate:"required,min=2"`
	BusinessType string `json:"business_type" validate:"required,oneof=beauty dental wellness barbershop clinic"`
	City         string `json:"city"`
	Country      string `json:"country"       validate:"omitempty,len=2"`
	Timezone     string `json:"timezone"`
}

// RegisterResponse tras un registro exitoso.
type RegisterResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	User         UserDTO   `json:"user"`
	Tenant       TenantDTO `json:"tenant"`
}

// UpdateTenantProfileRequest actualiza el perfil público del negocio.
type UpdateTenantProfileRequest struct {
	Name     string `json:"name"    validate:"required,min=2,max=120"`
	Phone    string `json:"phone"   validate:"omitempty,max=20"`
	City     string `json:"city"    validate:"omitempty,max=100"`
	Country  string `json:"country" validate:"omitempty,len=2"`
	Timezone string `json:"timezone" validate:"omitempty,max=50"`
}

// UpdateUserProfileRequest actualiza el nombre del propietario.
// El email NO se puede cambiar aquí — está vinculado a Supabase Auth.
type UpdateUserProfileRequest struct {
	Name string `json:"name" validate:"required,min=2,max=120"`
}

// LoginRequest credenciales para autenticarse.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse tras un login exitoso.
type LoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	User         UserDTO   `json:"user"`
	Tenant       TenantDTO `json:"tenant"`
}

// RefreshResponse tras renovar el token.
type RefreshResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ── Professional ──────────────────────────────────────────────────────────────

// Professional es un empleado/profesional del negocio.
type Professional struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Name      string     `json:"name"`
	Specialty string     `json:"specialty,omitempty"`
	Bio       string     `json:"bio,omitempty"`
	AvatarURL string     `json:"avatar_url,omitempty"`
	Phone     string     `json:"phone,omitempty"`
	Email     string     `json:"email,omitempty"`
	Color     string     `json:"color"`
	IsActive   bool       `json:"is_active"`
	IsArchived bool       `json:"is_archived"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ProfessionalInput datos para crear/actualizar un profesional.
type ProfessionalInput struct {
	Name      string `json:"name"      validate:"required,min=2"`
	Specialty string `json:"specialty"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	Phone     string `json:"phone"     validate:"omitempty,min=8,max=20"`
	Email     string `json:"email"     validate:"omitempty,email"`
	Color     string `json:"color"     validate:"omitempty,len=7"`
	IsActive   *bool `json:"is_active"`
	IsArchived *bool `json:"is_archived"`
}

// ── Service ───────────────────────────────────────────────────────────────────

// Service es un servicio que ofrece el negocio.
type Service struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	DurationMin int       `json:"duration_min"`
	Price       *float64  `json:"price,omitempty"`
	Currency    string    `json:"currency"`
	BufferMin   int       `json:"buffer_min"`
	IsActive    bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// ServiceInput datos para crear/actualizar un servicio.
type ServiceInput struct {
	Name        string   `json:"name"         validate:"required,min=2"`
	Description string   `json:"description"`
	DurationMin int      `json:"duration_min" validate:"required,min=5,max=480"`
	Price       *float64 `json:"price"        validate:"omitempty,min=0"`
	Currency    string   `json:"currency"     validate:"omitempty,len=3"`
	BufferMin   int      `json:"buffer_min"   validate:"min=0,max=120"`
	SortOrder   int      `json:"sort_order"`
	IsActive    *bool    `json:"is_active"`
}

// ── Schedule ──────────────────────────────────────────────────────────────────

// Schedule define el horario semanal de un profesional.
type Schedule struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ProfessionalID uuid.UUID `json:"professional_id"`
	DayOfWeek      int       `json:"day_of_week"` // 0=Dom, 1=Lun, ..., 6=Sáb
	StartTime      string    `json:"start_time"`  // "HH:MM"
	EndTime        string    `json:"end_time"`    // "HH:MM"
	IsActive       bool      `json:"is_active"`
}

// ScheduleBlock bloqueo específico de tiempo (vacaciones, festivos, etc.)
type ScheduleBlock struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ProfessionalID *uuid.UUID `json:"professional_id,omitempty"`
	StartsAt       time.Time  `json:"starts_at"`
	EndsAt         time.Time  `json:"ends_at"`
	Reason         string     `json:"reason,omitempty"`
	IsRecurring    bool  `json:"is_recurring"`
	RecurrenceDays []int `json:"recurrence_days,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ── Customer ──────────────────────────────────────────────────────────────────

// Customer es un cliente del negocio.
type Customer struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	Name              string     `json:"name"`
	Phone             string     `json:"phone,omitempty"`
	Email             string     `json:"email,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	Tags              []string   `json:"tags"`
	WaOptIn           bool       `json:"wa_opt_in"`
	TotalVisits       int        `json:"total_visits"`
	StageID           *uuid.UUID `json:"stage_id,omitempty"`
	LastVisitAt       *time.Time `json:"last_visit_at,omitempty"`
	NextRecallAt      *time.Time `json:"next_recall_at,omitempty"`
	LifetimeValue     float64    `json:"lifetime_value"`
	AcquisitionSource string     `json:"acquisition_source,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// ── Appointment ───────────────────────────────────────────────────────────────

// Appointment es una cita del negocio.
type Appointment struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	CustomerID         uuid.UUID  `json:"customer_id"`
	ProfessionalID     uuid.UUID  `json:"professional_id"`
	ServiceID          uuid.UUID  `json:"service_id"`
	TreatmentID        *uuid.UUID `json:"treatment_id,omitempty"`
	StartsAt           time.Time  `json:"starts_at"`
	EndsAt             time.Time  `json:"ends_at"`
	Status             string     `json:"status"`
	Source             string     `json:"source"`
	Price              *float64   `json:"price,omitempty"`
	Notes              string     `json:"notes,omitempty"`
	InternalNotes      string     `json:"internal_notes,omitempty"`
	ConfirmedAt        *time.Time `json:"confirmed_at,omitempty"`
	CancelledAt        *time.Time `json:"cancelled_at,omitempty"`
	CancellationReason string     `json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// AppointmentWithDetails incluye datos de customer, professional y service.
type AppointmentWithDetails struct {
	Appointment
	CustomerName     string `json:"customer_name"`
	CustomerPhone    string `json:"customer_phone"`
	ProfessionalName string `json:"professional_name"`
	ServiceName      string `json:"service_name"`
	ServiceDuration  int    `json:"service_duration_min"`
}

// CreateAppointmentRequest datos para crear una cita.
type CreateAppointmentRequest struct {
	ProfessionalID uuid.UUID  `json:"professional_id" validate:"required"`
	ServiceID      uuid.UUID  `json:"service_id"      validate:"required"`
	StartsAt       time.Time  `json:"starts_at"       validate:"required"`
	Notes          string     `json:"notes"`
	CustomerID     *uuid.UUID `json:"customer_id"`    // dashboard: cliente existente
	CustomerName   string     `json:"customer_name"`  // booking público: nuevo cliente
	CustomerPhone  string     `json:"customer_phone"` // booking público
	CustomerEmail  string     `json:"customer_email"` // booking público
	Source         string     `json:"source"`         // "dashboard","web","whatsapp","api"
	// Si está presente, materializa automáticamente una treatment_session
	// vinculada al tratamiento (status='pending', appointment_id=appt.ID).
	TreatmentID *uuid.UUID `json:"treatment_id"`
}

// UpdateAppointmentRequest actualiza estado/notas de una cita.
type UpdateAppointmentRequest struct {
	Status             string `json:"status"              validate:"omitempty,oneof=confirmed cancelled completed no_show"`
	Notes              string `json:"notes"`
	InternalNotes      string `json:"internal_notes"`
	CancellationReason string `json:"cancellation_reason"`
}

// RescheduleRequest datos para reagendar una cita.
type RescheduleRequest struct {
	ProfessionalID *uuid.UUID `json:"professional_id"`
	ServiceID      *uuid.UUID `json:"service_id"`
	StartsAt       time.Time  `json:"starts_at" validate:"required"`
}

// ── Availability ──────────────────────────────────────────────────────────────

// TimeSlot es un slot de tiempo disponible para reservar.
type TimeSlot struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

// AvailabilityQuery parámetros para consultar disponibilidad.
type AvailabilityQuery struct {
	ProfessionalID uuid.UUID `json:"professional_id" validate:"required"`
	ServiceID      uuid.UUID `json:"service_id"      validate:"required"`
	Date           string    `json:"date"            validate:"required"` // "YYYY-MM-DD"
	Timezone       string    `json:"timezone"`                            // ej: "America/Santo_Domingo"
}

// AppointmentListQuery filtros para listar citas con paginación.
type AppointmentListQuery struct {
	DateFrom       string     `json:"date_from"       validate:"required"`
	DateTo         string     `json:"date_to"         validate:"required"`
	Timezone       string     `json:"timezone"`
	ProfessionalID *uuid.UUID `json:"professional_id"`
	ServiceID      *uuid.UUID `json:"service_id"`
	Status         string     `json:"status"`
	Search         string     `json:"search"`
	SortBy         string     `json:"sort_by"`  // date, time, customer_name
	SortDir        string     `json:"sort_dir"` // asc, desc
	Page           int        `json:"page"`
	PerPage        int        `json:"per_page"`
}

// PaginatedAppointments resultado paginado de citas.
type PaginatedAppointments struct {
	Data       []*AppointmentWithDetails `json:"data"`
	Pagination Pagination                `json:"pagination"`
}

// Pagination metadatos de paginación.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ── Public booking ────────────────────────────────────────────────────────────

// ServiceProfessionalLink mapea qué profesional ofrece qué servicio.
type ServiceProfessionalLink struct {
	ServiceID      uuid.UUID `json:"service_id"`
	ProfessionalID uuid.UUID `json:"professional_id"`
}

// PublicProfile vista pública de un negocio (sin auth).
type PublicProfile struct {
	Slug          string          `json:"slug"`
	Name          string          `json:"name"`
	BusinessType  string          `json:"business_type"`
	City          string          `json:"city,omitempty"`
	Country       string          `json:"country,omitempty"`
	Timezone      string          `json:"timezone"`
	Services             []*Service               `json:"services"`
	Professionals        []*Professional           `json:"professionals"`
	ServiceProfessionals []ServiceProfessionalLink `json:"service_professionals"`
	BookingIntroText   string `json:"booking_intro_text,omitempty"`
	BookingSuccessText string `json:"booking_success_text,omitempty"`
	BotName            string `json:"bot_name,omitempty"`
	BotGreeting        string `json:"bot_greeting,omitempty"`
	LogoURL            string `json:"logo_url,omitempty"`
	CoverURL           string `json:"cover_url,omitempty"`
	Description        string `json:"description,omitempty"`
}

// PublicAppointment vista simplificada de una cita para endpoints públicos.
type PublicAppointment struct {
	ID               uuid.UUID `json:"id"`
	ServiceID        uuid.UUID `json:"service_id"`
	ServiceName      string    `json:"service_name"`
	ProfessionalID   uuid.UUID `json:"professional_id"`
	ProfessionalName string    `json:"professional_name"`
	StartsAt         time.Time `json:"starts_at"`
	EndsAt           time.Time `json:"ends_at"`
	Status           string    `json:"status"`
}

// PublicCancelRequest cuerpo del request para cancelar una cita pública.
type PublicCancelRequest struct {
	Phone string `json:"phone" validate:"required"`
}

// PublicRescheduleRequest cuerpo del request para reagendar una cita pública.
type PublicRescheduleRequest struct {
	Phone    string    `json:"phone" validate:"required"`
	StartsAt time.Time `json:"starts_at" validate:"required"`
}

// ── WhatsApp / Conversaciones ─────────────────────────────────────────────────

// Conversation es una sesión de chat vía WhatsApp con un cliente.
type Conversation struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty"`
	WAPhone       string     `json:"wa_phone"`
	Status        string     `json:"status"` // active, handed_off, closed
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Message es un mensaje individual de una conversación.
type Message struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Role           string    `json:"role"` // user, assistant, system
	Content        string    `json:"content"`
	WAMessageID    string    `json:"wa_message_id,omitempty"`
	TokensUsed     int       `json:"tokens_used,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// WAInboundPayload es el mensaje publicado en wa.messages.inbound (RabbitMQ).
type WAInboundPayload struct {
	TenantID       string                   `json:"tenant_id"`
	TenantSlug     string                   `json:"tenant_slug"`
	ConversationID string                   `json:"conversation_id"`
	WAPhone        string                   `json:"wa_phone"`
	WAMessageID    string                   `json:"wa_message_id"`
	Content        string                   `json:"content"`
	MessageType    string                   `json:"message_type"` // text, audio, image
	Timestamp      string                   `json:"timestamp"`
	History        []map[string]interface{} `json:"history"`      // últimos N mensajes [{role, content}]
}

// WAOutboundPayload es el mensaje publicado en wa.messages.outbound por el AI Service.
type WAOutboundPayload struct {
	TenantID       string `json:"tenant_id"`
	TenantSlug     string `json:"tenant_slug"`
	ConversationID string `json:"conversation_id"`
	WAPhone        string `json:"wa_phone"`
	Content        string `json:"content"`
}

// ── Knowledge Base ────────────────────────────────────────────────────────────

// KnowledgeDocument es un documento de la base de conocimiento del tenant.
type KnowledgeDocument struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Category   string    `json:"category"` // services, pricing, faq, policies, team, location, promotions
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	IsActive   bool      `json:"is_active"`
	SourceType string    `json:"source_type"`        // text | file
	FileName   *string   `json:"file_name,omitempty"` // nombre original del archivo subido
	Status     string    `json:"status"`             // ready | processing | error
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// KnowledgeDocumentInput datos para crear/actualizar un documento de texto.
type KnowledgeDocumentInput struct {
	Category string `json:"category" validate:"required,oneof=services pricing faq policies team location promotions"`
	Title    string `json:"title"    validate:"required,min=2,max=255"`
	Content  string `json:"content"  validate:"required,min=10"`
	IsActive *bool  `json:"is_active"`
}

// KnowledgeUploadInput datos del formulario multipart para subir un archivo.
type KnowledgeUploadInput struct {
	Category string `form:"category" validate:"required,oneof=services pricing faq policies team location promotions"`
	Title    string `form:"title"    validate:"required,min=2,max=255"`
	IsActive *bool  `form:"is_active"`
}

// KnowledgeVectorizePayload mensaje publicado en la cola knowledge.vectorize.
// Para texto manual: Content tiene el texto, FileBytes/FileType vacíos.
// Para archivos: FileBytes contiene base64 del archivo, FileType indica el tipo.
type KnowledgeVectorizePayload struct {
	TenantID   string `json:"tenant_id"`
	DocumentID string `json:"document_id"`
	Content    string `json:"content"`    // texto directo (flujo manual)
	FileBytes  string `json:"file_bytes"` // base64 del archivo (flujo upload)
	FileType   string `json:"file_type"`  // pdf | docx | xlsx (flujo upload)
}

// ── Chatbot Config ───────────────────────────────────────────────────────────

// ChatbotConfig configuración del chatbot IA de un tenant.
type ChatbotConfig struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	BotName            string     `json:"bot_name"`
	BotGreeting        string     `json:"bot_greeting"`
	Tone               string     `json:"tone"`
	CustomInstructions string     `json:"custom_instructions"`
	TemplateID         *string    `json:"template_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// UpdateChatbotConfigInput campos opcionales para PATCH del config.
type UpdateChatbotConfigInput struct {
	BotName            *string `json:"bot_name"             validate:"omitempty,max=30"`
	BotGreeting        *string `json:"bot_greeting"         validate:"omitempty,max=500"`
	Tone               *string `json:"tone"                 validate:"omitempty,oneof=friendly professional premium casual"`
	CustomInstructions *string `json:"custom_instructions"  validate:"omitempty,max=1000"`
	TemplateID         *string `json:"template_id"          validate:"omitempty,max=30"`
}

// ChatbotTestRequest mensaje de prueba para el chatbot.
type ChatbotTestRequest struct {
	Message string `json:"message" validate:"required,min=1,max=2000"`
}

// ChatbotTestResponse respuesta síncrona del chatbot en modo test.
type ChatbotTestResponse struct {
	Response         string   `json:"response"`
	IntentDetected   string   `json:"intent_detected"`
	RAGSourcesUsed   []string `json:"rag_sources_used"`
	ProcessingTimeMs int64    `json:"processing_time_ms"`
}

// ChatbotValidateResponse resultado de la validación IA del chatbot.
type ChatbotValidateResponse struct {
	TotalQuestions int                       `json:"total_questions"`
	Passed         int                       `json:"passed"`
	Results        []ChatbotValidationResult `json:"results"`
}

// ChatbotValidationResult resultado de una pregunta de validación individual.
type ChatbotValidationResult struct {
	Question   string `json:"question"`
	Response   string `json:"response"`
	Passed     bool   `json:"passed"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ── Recordatorios ─────────────────────────────────────────────────────────────

// ReminderJob representa una cita que necesita recordatorio.
type ReminderJob struct {
	AppointmentID    uuid.UUID `json:"appointment_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	CustomerID       uuid.UUID `json:"customer_id"`
	TenantSlug       string    `json:"tenant_slug"`
	TenantTimezone   string    `json:"tenant_timezone"`
	CustomerName     string    `json:"customer_name"`
	CustomerPhone    string    `json:"customer_phone"`
	ProfessionalName string    `json:"professional_name"`
	ServiceName      string    `json:"service_name"`
	StartsAt         time.Time `json:"starts_at"`
	Type             string    `json:"type"` // reminder_24h, reminder_2h
}

// NotificationLog registro de mensaje enviado.
type NotificationLog struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	AppointmentID *uuid.UUID `json:"appointment_id,omitempty"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty"`
	Type          string     `json:"type"`
	WAMessageID   string     `json:"wa_message_id,omitempty"`
	Status        string     `json:"status"` // sent, delivered, failed
	Content       string     `json:"content"`
	SentAt        time.Time  `json:"sent_at"`
	ErrorMessage  string     `json:"error_message,omitempty"`
}

// ── CRM Pipeline ──────────────────────────────────────────────────────────────

// PipelineStage es una etapa del pipeline CRM del tenant.
type PipelineStage struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	Name             string    `json:"name"`
	Position         int       `json:"position"`
	Color            string    `json:"color"`
	IsDefault        bool      `json:"is_default"`
	AutoRulesEnabled bool      `json:"auto_rules_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

// PipelineStageInput datos para crear/actualizar una etapa.
type PipelineStageInput struct {
	Name             string `json:"name"     validate:"required,min=2,max=100"`
	Position         int    `json:"position" validate:"min=0"`
	Color            string `json:"color"    validate:"omitempty,len=7"`
	IsDefault        *bool  `json:"is_default"`
	AutoRulesEnabled *bool  `json:"auto_rules_enabled"`
}

// ReorderStageInput reordena una etapa a una posicion nueva.
type ReorderStageInput struct {
	StageID  uuid.UUID `json:"stage_id"  validate:"required"`
	Position int       `json:"position"  validate:"min=0"`
}

// ── Treatments ────────────────────────────────────────────────────────────────

// Treatment es un plan de tratamiento asociado a un cliente.
type Treatment struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	CustomerID        uuid.UUID  `json:"customer_id"`
	ProfessionalID    uuid.UUID  `json:"professional_id"`
	Name              string     `json:"name"`
	TreatmentType     string     `json:"treatment_type"`
	Status            string     `json:"status"`
	TotalSessions     *int       `json:"total_sessions,omitempty"`
	CompletedSessions int        `json:"completed_sessions"`
	EstimatedCost     *float64   `json:"estimated_cost,omitempty"`
	PaidAmount        float64    `json:"paid_amount"`
	Currency          string     `json:"currency"`
	ToothNumbers      []int      `json:"tooth_numbers,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	NextSessionAt     *time.Time `json:"next_session_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TreatmentInput datos para crear/actualizar un tratamiento.
type TreatmentInput struct {
	CustomerID     uuid.UUID `json:"customer_id"     validate:"required"`
	ProfessionalID uuid.UUID `json:"professional_id" validate:"required"`
	Name           string    `json:"name"            validate:"required,min=2,max=255"`
	TreatmentType  string    `json:"treatment_type"  validate:"required,oneof=ortodoncia endodoncia implante protesis cirugia periodoncia estetica general"`
	TotalSessions  *int      `json:"total_sessions"  validate:"omitempty,min=1"`
	EstimatedCost  *float64  `json:"estimated_cost"  validate:"omitempty,min=0"`
	Currency       string    `json:"currency"        validate:"omitempty,len=3"`
	ToothNumbers   []int     `json:"tooth_numbers"`
	Notes          string    `json:"notes"`
}

// UpdateTreatmentStatusInput datos para cambiar el estado de un tratamiento.
type UpdateTreatmentStatusInput struct {
	Status string `json:"status" validate:"required,oneof=proposed accepted in_progress completed abandoned"`
}

// TreatmentSession representa una sesión individual dentro de un tratamiento.
type TreatmentSession struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	TreatmentID     uuid.UUID  `json:"treatment_id"`
	ProfessionalID  uuid.UUID  `json:"professional_id"`
	Status          string     `json:"status"`
	ScheduledAt     time.Time  `json:"scheduled_at"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ProceduresDone  string     `json:"procedures_done,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	PaidInSession   *float64   `json:"paid_in_session,omitempty"`
	Currency        string     `json:"currency,omitempty"`
	NextSessionAt   *time.Time `json:"next_session_at,omitempty"`
	// NULL si fue creada manualmente (registro retroactivo);
	// poblado si fue materializada desde un appointment con treatment_id.
	AppointmentID *uuid.UUID `json:"appointment_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// JOIN expandido
	ProfessionalName string `json:"professional_name,omitempty"`
}

// TreatmentSessionInput datos para crear una sesión.
type TreatmentSessionInput struct {
	ProfessionalID  uuid.UUID  `json:"professional_id"  validate:"required"`
	Status          string     `json:"status"           validate:"required,oneof=pending completed"`
	ScheduledAt     time.Time  `json:"scheduled_at"     validate:"required"`
	DurationMinutes *int       `json:"duration_minutes" validate:"omitempty,min=1"`
	ProceduresDone  string     `json:"procedures_done"`
	Notes           string     `json:"notes"`
	PaidInSession   *float64   `json:"paid_in_session"  validate:"omitempty,min=0"`
	Currency        string     `json:"currency"         validate:"omitempty,len=3"`
	NextSessionAt   *time.Time `json:"next_session_at"`
}

// UpdateTreatmentSessionInput datos para actualizar una sesión.
type UpdateTreatmentSessionInput struct {
	Status          string     `json:"status"           validate:"required,oneof=pending completed cancelled"`
	DurationMinutes *int       `json:"duration_minutes" validate:"omitempty,min=1"`
	ProceduresDone  string     `json:"procedures_done"`
	Notes           string     `json:"notes"`
	PaidInSession   *float64   `json:"paid_in_session"  validate:"omitempty,min=0"`
	Currency        string     `json:"currency"         validate:"omitempty,len=3"`
	NextSessionAt   *time.Time `json:"next_session_at"`
}

// TreatmentListQuery filtros para listar tratamientos.
type TreatmentListQuery struct {
	CustomerID     *uuid.UUID `json:"customer_id"`
	ProfessionalID *uuid.UUID `json:"professional_id"`
	Status         string     `json:"status"`
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

// Task es una tarea asignable a un profesional.
type Task struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	AssignedTo    *uuid.UUID `json:"assigned_to,omitempty"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty"`
	AppointmentID *uuid.UUID `json:"appointment_id,omitempty"`
	TreatmentID   *uuid.UUID `json:"treatment_id,omitempty"`
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	Status        string     `json:"status"`
	DueAt         *time.Time `json:"due_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Source        string     `json:"source"`
	RuleID        *uuid.UUID `json:"rule_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TaskInput datos para crear/actualizar una tarea.
type TaskInput struct {
	AssignedTo    *uuid.UUID `json:"assigned_to"`
	CustomerID    *uuid.UUID `json:"customer_id"`
	AppointmentID *uuid.UUID `json:"appointment_id"`
	TreatmentID   *uuid.UUID `json:"treatment_id"`
	Title         string     `json:"title"       validate:"required,min=2,max=255"`
	Description   string     `json:"description"`
	DueAt         *time.Time `json:"due_at"`
}

// TaskListQuery filtros para listar tareas.
type TaskListQuery struct {
	AssignedTo *uuid.UUID `json:"assigned_to"`
	CustomerID *uuid.UUID `json:"customer_id"`
	Status     string     `json:"status"`
	DueBefore  *time.Time `json:"due_before"`
}

// ── Rules ─────────────────────────────────────────────────────────────────────

// RuleCondition es una condicion que debe cumplirse para que la regla se ejecute.
type RuleCondition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// RuleAction es una accion a ejecutar cuando la regla se dispara.
type RuleAction struct {
	Type     string         `json:"type"`
	Template string         `json:"template"`
	Params   map[string]any `json:"params"`
}

// RuleTriggerSchedule configuracion para reglas temporales.
type RuleTriggerSchedule struct {
	IntervalDays   int    `json:"interval_days"`
	ReferenceField string `json:"reference_field"`
}

// Rule es una regla de automatizacion CRM.
type Rule struct {
	ID              uuid.UUID            `json:"id"`
	TenantID        uuid.UUID            `json:"tenant_id"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	TriggerType     string               `json:"trigger_type"`
	TriggerEvent    string               `json:"trigger_event,omitempty"`
	TriggerSchedule *RuleTriggerSchedule `json:"trigger_schedule,omitempty"`
	Conditions      []RuleCondition      `json:"conditions"`
	Actions         []RuleAction         `json:"actions"`
	IsActive        bool                 `json:"is_active"`
	IsTemplate      bool                 `json:"is_template"`
	TemplateKey     string               `json:"template_key,omitempty"`
	CooldownHours   int                  `json:"cooldown_hours"`
	Priority        int                  `json:"priority"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// RuleInput datos para crear/actualizar una regla.
type RuleInput struct {
	Name            string               `json:"name"            validate:"required,min=2,max=255"`
	Description     string               `json:"description"`
	TriggerType     string               `json:"trigger_type"    validate:"required,oneof=event temporal"`
	TriggerEvent    string               `json:"trigger_event"   validate:"omitempty"`
	TriggerSchedule *RuleTriggerSchedule `json:"trigger_schedule"`
	Conditions      []RuleCondition      `json:"conditions"`
	Actions         []RuleAction         `json:"actions"         validate:"required,min=1"`
	IsActive        *bool                `json:"is_active"`
	CooldownHours   int                  `json:"cooldown_hours"  validate:"min=0"`
	Priority        int                  `json:"priority"        validate:"min=0"`
}

// RuleExecution es el registro de una ejecucion de regla.
type RuleExecution struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	RuleID             uuid.UUID       `json:"rule_id"`
	CustomerID         *uuid.UUID      `json:"customer_id,omitempty"`
	TriggeredAt        time.Time       `json:"triggered_at"`
	TriggerEvent       string          `json:"trigger_event,omitempty"`
	ConditionsSnapshot []RuleCondition `json:"conditions_snapshot,omitempty"`
	ActionsResult      []RuleAction    `json:"actions_result,omitempty"`
	Status             string          `json:"status"`
	ErrorMessage       string          `json:"error_message,omitempty"`
}

// ── Rule Events ──────────────────────────────────────────────────────────────

// RuleEvent representa un evento de dominio que puede disparar reglas de automatización.
type RuleEvent struct {
	TenantID   uuid.UUID      `json:"tenant_id"`
	EventType  string         `json:"event_type"`
	CustomerID uuid.UUID      `json:"customer_id"`
	EntityID   uuid.UUID      `json:"entity_id"`
	EntityType string         `json:"entity_type"`
	Payload    map[string]any `json:"payload"`
	Timestamp  time.Time      `json:"timestamp"`
}

// ── CRM Metrics ─────────────────────────────────────────────────────────────

// CRMMetrics agrega metricas del CRM para el dashboard.
type CRMMetrics struct {
	RulesFired30d     int                  `json:"rules_fired_30d"`
	RulesSuccessRate  float64              `json:"rules_success_rate"`
	ActiveTreatments  int                  `json:"active_treatments"`
	PendingTasks      int                  `json:"pending_tasks"`
	CustomersPerStage []StageCustomerCount `json:"customers_per_stage"`
	TopRules          []TopRuleMetric      `json:"top_rules"`
}

// StageCustomerCount cantidad de clientes en una etapa del pipeline.
type StageCustomerCount struct {
	StageID   uuid.UUID `json:"stage_id"`
	StageName string    `json:"stage_name"`
	Color     string    `json:"color"`
	Count     int       `json:"count"`
}

// TopRuleMetric regla con mas ejecuciones en los ultimos 30 dias.
type TopRuleMetric struct {
	RuleID     uuid.UUID `json:"rule_id"`
	RuleName   string    `json:"rule_name"`
	Executions int       `json:"executions"`
}

// ── Waitlist (pre-launch) ─────────────────────────────────────────────────────

// WaitlistSignup representa un lead capturado en la landing pre-launch.
// La tabla `waitlist_signups` es global (sin tenant_id, sin RLS) porque los
// visitantes dejan sus datos antes de tener cuenta.
type WaitlistSignup struct {
	ID           uuid.UUID `json:"id"             db:"id"`
	Email        string    `json:"email"          db:"email"`
	BusinessName string    `json:"business_name"  db:"business_name"`
	IPAddress    string    `json:"ip_address,omitempty"  db:"ip_address"`
	UserAgent    string    `json:"user_agent,omitempty"  db:"user_agent"`
	CreatedAt    time.Time `json:"created_at"     db:"created_at"`
}

// JoinWaitlistInput datos crudos recibidos desde el handler para sumar un lead.
// La validación de email/business_name se hace en el service.
type JoinWaitlistInput struct {
	Email        string `json:"email"         validate:"required,email,max=254"`
	BusinessName string `json:"business_name" validate:"required,min=2,max=120"`
	// Capturados por el handler desde el request HTTP, no vienen en el body.
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

// ── Dental Notification Jobs (Plane #34) ──────────────────────────────────────

// PostOpJob datos necesarios para enviar un mensaje post-op 48h después
// de completar una sesión de tratamiento. Producido por el worker de
// dental notifications (sin RLS — query cross-tenant filtrada por módulo).
type PostOpJob struct {
	SessionID        uuid.UUID
	TreatmentID      uuid.UUID
	TenantID         uuid.UUID
	TenantSlug       string
	CustomerID       uuid.UUID
	CustomerName     string
	CustomerPhone    string
	TreatmentName    string
	TreatmentType    string
	ProfessionalName string
	CompletedAt      time.Time
}

// RecallJob datos necesarios para enviar un recall 6 meses después de
// completar un tratamiento.
type RecallJob struct {
	TreatmentID   uuid.UUID
	TenantID      uuid.UUID
	TenantSlug    string
	CustomerID    uuid.UUID
	CustomerName  string
	CustomerPhone string
	TreatmentName string
	TreatmentType string
	CompletedAt   time.Time
}

// CustomerTreatmentSummary resumen liviano de un tratamiento del cliente para
// devolver vía endpoint público al AI service (tool list_my_treatments).
type CustomerTreatmentSummary struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	TreatmentType     string     `json:"treatment_type"`
	Status            string     `json:"status"`
	CompletedSessions int        `json:"completed_sessions"`
	TotalSessions     *int       `json:"total_sessions,omitempty"`
	NextSessionAt     *time.Time `json:"next_session_at,omitempty"`
	ProfessionalName  string     `json:"professional_name,omitempty"`
}

// ── Tenant Modules ────────────────────────────────────────────────────────────

// Module keys registrados. Agregar aquí cualquier nuevo módulo activable.
const (
	ModuleDental = "dental"
)

// TenantModule representa la activación de un módulo opcional para un tenant.
// Audita las transiciones enabled/disabled con timestamps.
type TenantModule struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	ModuleKey     string     `json:"module_key"`
	Enabled       bool       `json:"enabled"`
	ActivatedAt   time.Time  `json:"activated_at"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
