// Wiring de rutas del Core API.
// Extraído de main.go para permitir testear el wiring (en particular, el gating
// del módulo dental aplicado a treatments y clinical notes/files).
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/citaspot/api/internal/config"
	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/handler"
	"github.com/citaspot/api/internal/middleware"
	"github.com/citaspot/api/internal/seed"
)

// RouteDeps agrupa todas las dependencias requeridas para montar las rutas.
// Permite reemplazar la cadena de middlewares de auth para tests.
type RouteDeps struct {
	// Infra
	Cfg  *config.Config
	Pool *pgxpool.Pool
	Rdb  *redis.Client

	// Repos usados directamente en closures o middlewares
	AuthRepo         domain.AuthRepository
	TenantModuleRepo domain.TenantModuleRepository

	// Services usados directamente en closures
	ProfSvc domain.ProfessionalSvc

	// Handlers
	AuthHandler             *handler.AuthHandler
	ProfHandler             *handler.ProfessionalHandler
	SvcHandler              *handler.ServiceHandler
	ApptHandler             *handler.AppointmentHandler
	PubHandler              *handler.PublicHandler
	WaHandler               *handler.WhatsAppHandler
	KnowledgeHandler        *handler.KnowledgeHandler
	CustomerHandler         *handler.CustomerHandler
	SettingsHandler         *handler.SettingsHandler
	BlockHandler            *handler.ScheduleBlockHandler
	BillingHandler          *handler.BillingHandler
	PipelineHandler         *handler.PipelineStageHandler
	TreatmentHandler        *handler.TreatmentHandler
	TreatmentSessionHandler *handler.TreatmentSessionHandler
	TaskHandler             *handler.TaskHandler
	RuleHandler             *handler.RuleHandler
	CrmHandler              *handler.CRMHandler
	BrandingHandler         *handler.BrandingHandler
	ClinicalNoteHandler     *handler.ClinicalNoteHandler
	ClinicalFileHandler     *handler.ClinicalFileHandler
	ChatbotHandler          *handler.ChatbotHandler
	WaitlistHandler         *handler.WaitlistHandler
	RealtimeHandler         *handler.RealtimeHandler

	// Override del stack de auth para tests.
	// Si es nil, se construye la cadena real: JWT + Tenant + PlanCheck.
	// Si está seteado, se aplican estos middlewares al grupo /protected.
	AuthMiddlewares []fiber.Handler

	// Override del gate del módulo dental para tests.
	// Si es nil, se construye con RequireModule + TenantModuleRepo.
	// Si está seteado, se aplica este handler donde iría el dental gate.
	ModuleGate fiber.Handler
}

// SetupRoutes monta todas las rutas de negocio sobre app.
// No incluye health checks, swagger ni CORS — esos quedan en main.go porque
// no forman parte del wiring de negocio testeable.
func SetupRoutes(app *fiber.App, deps *RouteDeps) {
	api := app.Group("/api/v1")
	// Preflight CORS: el navegador envía OPTIONS antes de POST/GET
	api.Options("/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	// ── Rate limiting en rutas públicas sensibles ─────────────────────────────
	// Auth: 10 req/min por IP — protege contra brute force
	auth := api.Group("/auth", limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "auth:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":    "RATE_LIMIT",
				"message": "Demasiados intentos. Intenta de nuevo en un minuto.",
			})
		},
	}))
	auth.Post("/register", deps.AuthHandler.Register)
	auth.Post("/login", deps.AuthHandler.Login)
	auth.Post("/refresh", deps.AuthHandler.RefreshToken)

	// WhatsApp webhook: 300 req/min — Evolution envía ráfagas de eventos
	api.Post("/whatsapp/webhook", limiter.New(limiter.Config{
		Max:        300,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "wa_webhook:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusTooManyRequests)
		},
	}), deps.WaHandler.Webhook)

	// Stripe webhook: 50 req/min
	api.Post("/billing/webhook", limiter.New(limiter.Config{
		Max:        50,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "stripe_webhook:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusTooManyRequests)
		},
	}), deps.BillingHandler.Webhook)

	// Booking público (sin JWT, por slug)
	pub := api.Group("/public")
	pub.Get("/:slug", deps.PubHandler.GetProfile)
	pub.Get("/:slug/availability", deps.PubHandler.GetAvailability)
	pub.Post("/:slug/book", deps.PubHandler.Book)
	pub.Get("/:slug/my-appointments", deps.PubHandler.ListMyAppointments)
	pub.Get("/:slug/my-treatments", deps.PubHandler.ListMyTreatments)
	pub.Post("/:slug/appointments/:id/cancel", deps.PubHandler.CancelAppointment)
	pub.Post("/:slug/appointments/:id/reschedule", deps.PubHandler.RescheduleAppointment)

	// Lista de espera pre-launch (sin auth, sin tenant). Rate limit estricto
	// por IP para prevenir abuse de bots que rellenen la tabla.
	pub.Post("/waitlist", limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "waitlist:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":    "RATE_LIMIT",
				"message": "Demasiados intentos. Intenta de nuevo en un minuto.",
			})
		},
	}), deps.WaitlistHandler.Join)

	// ── Rutas protegidas (JWT + tenant + plan check) ───────────────────────────
	// Rate limit general: 120 req/min por IP en todas las rutas autenticadas
	apiLimiter := limiter.New(limiter.Config{
		Max:        120,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "api:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":    "RATE_LIMIT",
				"message": "Demasiadas solicitudes. Intenta en un momento.",
			})
		},
	})

	// Cadena de auth: producción usa JWT+Tenant+PlanCheck; tests pueden inyectar mocks.
	authChain := deps.AuthMiddlewares
	if authChain == nil {
		authChain = []fiber.Handler{
			middleware.JWTMiddleware(deps.Cfg.JWTSecret, deps.Cfg.SupabaseURL),
			middleware.TenantMiddleware(deps.AuthRepo, deps.Pool),
			middleware.PlanCheckMiddleware(),
		}
	}
	protectedMW := append([]fiber.Handler{apiLimiter}, authChain...)
	protected := api.Group("", protectedMW...)

	protected.Get("/me", func(c *fiber.Ctx) error {
		tenantID := middleware.TenantIDFromContext(c)
		modules, err := deps.TenantModuleRepo.ListActiveKeys(c.Context(), tenantID)
		if err != nil {
			modules = []string{}
		}
		return c.JSON(fiber.Map{
			"user":            middleware.UserFromContext(c),
			"tenant":          middleware.TenantFromContext(c),
			"enabled_modules": modules,
		})
	})

	// Marca el onboarding del tenant como completado.
	protected.Post("/onboarding/complete", func(c *fiber.Ctx) error {
		tenantID := middleware.TenantIDFromContext(c)
		if tenantID == uuid.Nil {
			return fiber.NewError(403, "tenant no identificado")
		}
		if err := deps.AuthRepo.CompleteOnboarding(c.Context(), tenantID); err != nil {
			return fiber.NewError(500, "error interno")
		}

		// Auto-asigna servicios al primer profesional para que /book/{slug} funcione tras onboarding.
		if n, err := deps.ProfSvc.AutoAssignAllServicesToFirstProfessional(c.Context(), tenantID); err != nil {
			slog.Warn("onboarding: auto-asignación falló", "tenant_id", tenantID, "error", err)
		} else if n > 0 {
			slog.Info("onboarding: servicios auto-asignados", "tenant_id", tenantID, "count", n)
		}

		// Auto-seed CRM por business_type — pipeline aplica a todos los verticales
		// con template; rule templates y knowledge siguen siendo dental-only.
		tenant := middleware.TenantFromContext(c)
		if tenant != nil {
			bt := tenant.BusinessType
			go func() {
				bgCtx := context.Background()
				if seeded, err := seed.SeedPipelineForBusinessType(bgCtx, deps.Pool, tenantID, bt); err != nil {
					slog.Warn("onboarding: error seeding pipeline", "tenant_id", tenantID, "business_type", bt, "error", err)
				} else if len(seeded) > 0 {
					slog.Info("onboarding: pipeline seeded", "tenant_id", tenantID, "business_type", bt, "stages", len(seeded))
				}
				// Templates genericos vertical-agnosticos (post-cita, etc) — corren para todos los tenants.
				if err := seed.SeedGenericRuleTemplates(bgCtx, deps.Pool); err != nil {
					slog.Warn("onboarding: error seeding generic rule templates", "error", err)
				}
				// Copiar reglas de notificación de citas al tenant nuevo
				if err := seed.SeedAppointmentRulesForTenant(bgCtx, deps.Pool, tenantID); err != nil {
					slog.Warn("onboarding: error seeding appointment rules", "tenant_id", tenantID, "error", err)
				} else {
					slog.Info("onboarding: appointment notification rules seeded", "tenant_id", tenantID)
				}
				if bt == "dental" {
					// Rule templates son globales (idempotent) — safe to call multiple times
					if err := seed.SeedDentalRuleTemplates(bgCtx, deps.Pool); err != nil {
						slog.Warn("onboarding: error seeding dental rule templates", "error", err)
					}
					if err := seed.SeedDentalKnowledge(bgCtx, deps.Pool, tenantID); err != nil {
						slog.Warn("onboarding: error seeding dental knowledge", "tenant_id", tenantID, "error", err)
					} else {
						slog.Info("onboarding: dental knowledge seeded", "tenant_id", tenantID)
					}
				}
			}()
		}

		return c.JSON(fiber.Map{"ok": true})
	})

	profs := protected.Group("/professionals")
	profs.Get("/", deps.ProfHandler.List)
	profs.Post("/", deps.ProfHandler.Create)
	profs.Get("/:id", deps.ProfHandler.GetByID)
	profs.Patch("/:id", deps.ProfHandler.Update)
	profs.Get("/:id/schedule", deps.ProfHandler.GetSchedule)
	profs.Put("/:id/schedule", deps.ProfHandler.SetSchedule)
	profs.Get("/:id/services", deps.ProfHandler.ListServices)
	profs.Post("/:id/services/:serviceID", deps.ProfHandler.AssignService)
	profs.Delete("/:id/services/:serviceID", deps.ProfHandler.RemoveService)
	profs.Get("/:id/customers", deps.ProfHandler.ListCustomers)

	srvs := protected.Group("/services")
	srvs.Get("/", deps.SvcHandler.List)
	srvs.Post("/", deps.SvcHandler.Create)
	srvs.Get("/:id", deps.SvcHandler.GetByID)
	srvs.Patch("/:id", deps.SvcHandler.Update)
	srvs.Delete("/:id", deps.SvcHandler.Delete)

	appts := protected.Group("/appointments")
	appts.Get("/availability", deps.ApptHandler.Availability)
	appts.Get("/search", deps.ApptHandler.ListFiltered)
	appts.Get("/upcoming", deps.ApptHandler.Upcoming)
	appts.Get("/", deps.ApptHandler.List)
	appts.Post("/", deps.ApptHandler.Create)
	appts.Get("/:id", deps.ApptHandler.GetByID)
	appts.Patch("/:id", deps.ApptHandler.Update)
	appts.Delete("/:id/cancel", deps.ApptHandler.Cancel)
	appts.Patch("/:id/reschedule", deps.ApptHandler.Reschedule)

	knowledge := protected.Group("/knowledge")
	knowledge.Get("/", deps.KnowledgeHandler.List)
	knowledge.Post("/", deps.KnowledgeHandler.Create)
	knowledge.Post("/upload", deps.KnowledgeHandler.Upload)
	knowledge.Get("/:id", deps.KnowledgeHandler.Get)
	knowledge.Put("/:id", deps.KnowledgeHandler.Update)
	knowledge.Delete("/:id", deps.KnowledgeHandler.Delete)

	chatbot := protected.Group("/chatbot")
	chatbot.Get("/config", deps.ChatbotHandler.GetConfig)
	chatbot.Patch("/config", deps.ChatbotHandler.UpdateConfig)
	chatbot.Post("/test", deps.ChatbotHandler.Test)
	chatbot.Post("/validate", deps.ChatbotHandler.Validate)

	customers := protected.Group("/customers")
	customers.Get("/", deps.CustomerHandler.List)
	customers.Get("/:id", deps.CustomerHandler.GetByID)
	customers.Patch("/:id/stage", deps.CustomerHandler.UpdateStage)

	protected.Get("/whatsapp/status", deps.WaHandler.Status)
	protected.Get("/whatsapp/qr", deps.WaHandler.GetQR)
	protected.Post("/whatsapp/connect", deps.WaHandler.Connect)
	protected.Delete("/whatsapp/disconnect", deps.WaHandler.Disconnect)

	// Billing: checkout, subscription info, invoices y cancel requieren auth
	billing := protected.Group("/billing")
	billing.Post("/checkout", deps.BillingHandler.CreateCheckout)
	billing.Get("/subscription", deps.BillingHandler.Subscription)
	billing.Get("/invoices", deps.BillingHandler.Invoices)
	billing.Post("/cancel", deps.BillingHandler.CancelSubscription)

	// Settings
	protected.Get("/settings", deps.SettingsHandler.Get)
	protected.Patch("/settings", deps.SettingsHandler.Update)
	protected.Patch("/tenant/profile", deps.SettingsHandler.UpdateTenantProfile)
	protected.Patch("/me/profile", deps.SettingsHandler.UpdateMyProfile)

	// Tenant branding (logo + portada). Las rutas siempre se registran.
	// Si MinIO no está disponible, el handler devuelve 503 con mensaje descriptivo.
	branding := protected.Group("/tenant/branding")
	branding.Post("/logo", deps.BrandingHandler.UploadLogo)
	branding.Post("/cover", deps.BrandingHandler.UploadCover)
	branding.Delete("/:kind", deps.BrandingHandler.Remove)

	// Schedule Blocks
	blocks := protected.Group("/schedule-blocks")
	blocks.Post("/", deps.BlockHandler.Create)
	blocks.Get("/", deps.BlockHandler.List)
	blocks.Delete("/:id", deps.BlockHandler.Delete)

	// Pipeline Stages
	stages := protected.Group("/pipeline-stages")
	stages.Get("/", deps.PipelineHandler.List)
	stages.Post("/", deps.PipelineHandler.Create)
	stages.Post("/load-template", func(c *fiber.Ctx) error {
		tenantID := middleware.TenantIDFromContext(c)
		if tenantID == uuid.Nil {
			return fiber.NewError(403, "tenant no identificado")
		}
		tenant := middleware.TenantFromContext(c)
		if tenant == nil {
			return fiber.NewError(403, "tenant no identificado")
		}
		seeded, err := seed.SeedPipelineForBusinessType(c.Context(), deps.Pool, tenantID, tenant.BusinessType)
		if err != nil {
			slog.Error("pipeline load-template: failed", "tenant_id", tenantID, "error", err)
			return fiber.NewError(500, "error interno")
		}
		if len(seeded) == 0 {
			return fiber.NewError(409, "el pipeline ya tiene etapas o el vertical no tiene template")
		}
		return c.JSON(fiber.Map{"ok": true, "stages": seeded})
	})
	stages.Put("/reorder", deps.PipelineHandler.Reorder)
	stages.Get("/:id", deps.PipelineHandler.GetByID)
	stages.Patch("/:id", deps.PipelineHandler.Update)
	stages.Delete("/:id", deps.PipelineHandler.Delete)

	// Gate compartido del módulo dental — reusado en treatments y clinical notes/files.
	// En tests se puede inyectar deps.ModuleGate para verificar que aplica donde corresponde.
	dentalGate := deps.ModuleGate
	if dentalGate == nil {
		dentalGate = middleware.RequireModule(deps.TenantModuleRepo, domain.ModuleDental)
	}

	// Treatments — requiere módulo dental activo (Plane #31)
	treatments := protected.Group("/treatments", dentalGate)
	treatments.Get("/", deps.TreatmentHandler.List)
	treatments.Post("/", deps.TreatmentHandler.Create)
	treatments.Get("/:id", deps.TreatmentHandler.GetByID)
	treatments.Patch("/:id", deps.TreatmentHandler.Update)
	treatments.Patch("/:id/status", deps.TreatmentHandler.UpdateStatus)

	// Sesiones de tratamiento (heredan RequireModule del grupo treatments)
	treatmentSessions := treatments.Group("/:id/sessions")
	treatmentSessions.Get("/", deps.TreatmentSessionHandler.List)
	treatmentSessions.Post("/", deps.TreatmentSessionHandler.Create)
	treatmentSessions.Get("/:sid", deps.TreatmentSessionHandler.GetByID)
	treatmentSessions.Patch("/:sid", deps.TreatmentSessionHandler.Update)
	treatmentSessions.Delete("/:sid", deps.TreatmentSessionHandler.Delete)

	// Tasks
	tasks := protected.Group("/tasks")
	tasks.Get("/", deps.TaskHandler.List)
	tasks.Post("/", deps.TaskHandler.Create)
	tasks.Get("/:id", deps.TaskHandler.GetByID)
	tasks.Patch("/:id", deps.TaskHandler.Update)
	tasks.Post("/:id/complete", deps.TaskHandler.Complete)
	tasks.Post("/:id/dismiss", deps.TaskHandler.Dismiss)

	// Rules
	rules := protected.Group("/rules")
	rules.Get("/", deps.RuleHandler.List)
	rules.Post("/", deps.RuleHandler.Create)
	rules.Get("/:id", deps.RuleHandler.GetByID)
	rules.Patch("/:id", deps.RuleHandler.Update)
	rules.Delete("/:id", deps.RuleHandler.Delete)
	rules.Get("/:id/executions", deps.RuleHandler.ListExecutions)

	// Clinical Notes — requieren módulo dental activo
	appts.Post("/:id/clinical-note", dentalGate, deps.ClinicalNoteHandler.Create)
	appts.Get("/:id/clinical-note", dentalGate, deps.ClinicalNoteHandler.GetByAppointment)
	customers.Get("/:id/clinical-notes", dentalGate, deps.ClinicalNoteHandler.ListByCustomer)
	customers.Get("/:id/clinical-notes/:noteId", dentalGate, deps.ClinicalNoteHandler.GetByID)
	clinicalNotes := protected.Group("/clinical-notes", dentalGate)
	clinicalNotes.Patch("/:noteId", deps.ClinicalNoteHandler.Update)
	clinicalNotes.Delete("/:noteId", deps.ClinicalNoteHandler.Delete)
	clinicalNotes.Post("/:noteId/files/upload", deps.ClinicalFileHandler.Upload)
	clinicalNotes.Get("/:noteId/files", deps.ClinicalFileHandler.ListByNote)

	// Clinical Files — requieren módulo dental activo
	customers.Get("/:id/clinical-files", dentalGate, deps.ClinicalFileHandler.ListByCustomer)
	protected.Delete("/clinical-files/:fileId", dentalGate, deps.ClinicalFileHandler.Delete)

	// CRM Metrics
	protected.Get("/crm/metrics", deps.CrmHandler.Metrics)

	// Realtime (SSE) — Phase C: grupo dedicado con JWTMiddlewareWithQuery
	// porque EventSource no permite setear cabeceras custom (Authorization),
	// así que el token viaja como ?token=<jwt>. SEGURIDAD: el token en la URL
	// queda expuesto si se loguea — el access logger global salta /realtime/*
	// (ver main.go: `fiberlogger.New(...)` con `Next:`), y el scrubber de abajo
	// loguea SOLO method/path/ip/status sin query string.
	if deps.RealtimeHandler != nil {
		// Scrubber primero para que TODOS los requests (incluso 401/403) queden
		// registrados sin el query string ?token=...
		realtimeScrubber := func(c *fiber.Ctx) error {
			err := c.Next()
			slog.Info("realtime request",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("ip", c.IP()),
				slog.Int("status", c.Response().StatusCode()),
			)
			return err
		}
		realtime := api.Group("/realtime",
			realtimeScrubber,
			apiLimiter,
			middleware.JWTMiddlewareWithQuery(deps.Cfg.JWTSecret, deps.Cfg.SupabaseURL),
			middleware.TenantMiddleware(deps.AuthRepo, deps.Pool),
			middleware.PlanCheckMiddleware(),
		)
		realtime.Get("/appointments", deps.RealtimeHandler.Appointments)
	}
}
