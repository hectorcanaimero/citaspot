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
	Plan         string     `json:"plan"`
	PlanStatus   string     `json:"plan_status"`
	TrialEndsAt  *time.Time `json:"trial_ends_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
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
	ID           uuid.UUID  `json:"id"`
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	BusinessType string     `json:"business_type"`
	Plan         string     `json:"plan"`
	PlanStatus   string     `json:"plan_status"`
	TrialEndsAt  *time.Time `json:"trial_ends_at,omitempty"`
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
	Color     string     `json:"color"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}

// ProfessionalInput datos para crear/actualizar un profesional.
type ProfessionalInput struct {
	Name      string `json:"name"      validate:"required,min=2"`
	Specialty string `json:"specialty"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	Color     string `json:"color"     validate:"omitempty,len=7"`
	IsActive  *bool  `json:"is_active"`
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
	CreatedAt      time.Time  `json:"created_at"`
}

// ── Customer ──────────────────────────────────────────────────────────────────

// Customer es un cliente del negocio.
type Customer struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Tags        []string  `json:"tags"`
	WaOptIn     bool      `json:"wa_opt_in"`
	TotalVisits int       `json:"total_visits"`
	CreatedAt   time.Time `json:"created_at"`
}

// ── Appointment ───────────────────────────────────────────────────────────────

// Appointment es una cita del negocio.
type Appointment struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	CustomerID         uuid.UUID  `json:"customer_id"`
	ProfessionalID     uuid.UUID  `json:"professional_id"`
	ServiceID          uuid.UUID  `json:"service_id"`
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
}

// UpdateAppointmentRequest actualiza estado/notas de una cita.
type UpdateAppointmentRequest struct {
	Status             string `json:"status"              validate:"omitempty,oneof=confirmed cancelled completed no_show"`
	Notes              string `json:"notes"`
	InternalNotes      string `json:"internal_notes"`
	CancellationReason string `json:"cancellation_reason"`
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

// ── Public booking ────────────────────────────────────────────────────────────

// PublicProfile vista pública de un negocio (sin auth).
type PublicProfile struct {
	Slug          string          `json:"slug"`
	Name          string          `json:"name"`
	BusinessType  string          `json:"business_type"`
	City          string          `json:"city,omitempty"`
	Country       string          `json:"country,omitempty"`
	Services      []*Service      `json:"services"`
	Professionals []*Professional `json:"professionals"`
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

// ── Recordatorios ─────────────────────────────────────────────────────────────

// ReminderJob representa una cita que necesita recordatorio.
type ReminderJob struct {
	AppointmentID    uuid.UUID `json:"appointment_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
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
