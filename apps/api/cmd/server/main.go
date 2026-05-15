// Entrypoint del Core API de CitaSpot.
// Solo configuración y arranque — la lógica va en los paquetes internos.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // Embebe la base de datos de timezones en el binario (Alpine/scratch no la incluyen)

	"github.com/google/uuid"
	"github.com/gofiber/contrib/swagger"
	"github.com/joho/godotenv"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/citaspot/api/docs"
	"github.com/citaspot/api/internal/client/evolution"
	"github.com/citaspot/api/internal/client/rabbitmq"
	"github.com/citaspot/api/internal/client/storage"
	"github.com/citaspot/api/internal/config"
	"github.com/citaspot/api/internal/domain"
	"github.com/citaspot/api/internal/engine"
	"github.com/citaspot/api/internal/engine/actions"
	"github.com/citaspot/api/internal/handler"
	"github.com/citaspot/api/internal/logger"
	"github.com/citaspot/api/internal/middleware"
	"github.com/citaspot/api/internal/repository"
	"github.com/citaspot/api/internal/seed"
	"github.com/citaspot/api/internal/service"
	"github.com/citaspot/api/internal/worker"
)

// redocHTML es la página ReDoc que carga la spec desde /openapi.json.
const redocHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>CitaSpot API — ReDoc</title>
  <link href="https://cdn.jsdelivr.net/npm/redoc@2.2.0/bundles/redoc.standalone.css" rel="stylesheet">
</head>
<body>
  <div id="redoc-container"></div>
  <script src="https://cdn.jsdelivr.net/npm/redoc@2.2.0/bundles/redoc.standalone.js"></script>
  <script>
    Redoc.init('/openapi.json', {}, document.getElementById('redoc-container'));
  </script>
</body>
</html>`

func main() {
	// Carga .env si existe — útil cuando se corre el API local (no Docker).
	// Busca en el directorio actual y luego en la raíz del monorepo (../../.env).
	// En producción no hay .env y esto es no-op.
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../../.env")
	}

	cfg := config.Load()
	logger.Init(cfg.AppEnv)

	// Contexto con señales de apagado (Ctrl+C / SIGTERM)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// ── Base de datos ─────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Error conectando a PostgreSQL", "error", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("PostgreSQL no responde", "error", err)
	}
	slog.Info("PostgreSQL connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Fatal("Redis URL inválida", "error", err)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		// Redis es requerido para rate limiting e idempotencia — fatal en producción
		if cfg.AppEnv == "production" {
			logger.Fatal("Redis no responde", "error", err)
		}
		slog.Warn("Redis no disponible — rate limiting e idempotencia desactivados", "error", err)
		rdb = nil
	} else {
		slog.Info("Redis connected")
	}

	// ── Clientes externos ─────────────────────────────────────────────────────
	waClient := evolution.New(cfg.EvolutionAPIURL, cfg.EvolutionAPIKey, cfg.WebhookURL)

	publisher, pubErr := rabbitmq.New(cfg.RabbitMQURL)
	if pubErr != nil {
		slog.Warn("RabbitMQ no disponible (modo degradado)", "error", pubErr)
	} else {
		defer publisher.Close()
		slog.Info("RabbitMQ connected")
	}

	// ── Repositorios ──────────────────────────────────────────────────────────
	authRepo     := repository.NewAuthRepository(pool)
	profRepo     := repository.NewProfessionalRepository(pool)
	serviceRepo  := repository.NewServiceRepository(pool)
	scheduleRepo := repository.NewScheduleRepository(pool)
	customerRepo := repository.NewCustomerRepository(pool)
	apptRepo     := repository.NewAppointmentRepository(pool)
	convRepo      := repository.NewConversationRepository(pool)
	notifRepo     := repository.NewNotificationRepository(pool)
	reminderRepo  := repository.NewReminderRepository(pool)
	knowledgeRepo := repository.NewKnowledgeRepository(pool)
	pipelineRepo  := repository.NewPipelineStageRepository(pool)
	treatmentRepo        := repository.NewTreatmentRepository(pool)
	treatmentSessionRepo := repository.NewTreatmentSessionRepository(pool)
	taskRepo             := repository.NewTaskRepository(pool)
	ruleRepo      := repository.NewRuleRepository(pool)
	ruleExecRepo  := repository.NewRuleExecutionRepository(pool)
	crmMetricsRepo := repository.NewCRMMetricsRepository(pool)
	clinicalNoteRepo := repository.NewClinicalNoteRepository(pool)
	clinicalFileRepo := repository.NewClinicalFileRepository(pool)
	eventRepo        := repository.NewEventRepository(pool)
	chatbotConfigRepo := repository.NewChatbotConfigRepository(pool)

	// ── Servicios ─────────────────────────────────────────────────────────────
	authSvc    := service.NewAuthService(authRepo, cfg)
	profSvc    := service.NewProfessionalService(profRepo, scheduleRepo)
	serviceSvc := service.NewServiceSvc(serviceRepo)
	availSvc   := service.NewAvailabilityService(scheduleRepo, serviceRepo)
	apptSvc    := service.NewAppointmentSvc(apptRepo, serviceRepo, customerRepo, authRepo, waClient, notifRepo, publisher, eventRepo)
	publicSvc  := service.NewPublicSvc(authRepo, profRepo, serviceRepo, availSvc, apptSvc, customerRepo)

	var waSvc domain.WhatsAppSvc
	if publisher != nil {
		waSvc = service.NewWhatsAppSvc(authRepo, convRepo, customerRepo, publisher, eventRepo)
	}

	// publisher puede ser nil si RabbitMQ no está disponible (modo degradado)
	knowledgeSvc := service.NewKnowledgeSvc(knowledgeRepo, publisher)
	chatbotSvc   := service.NewChatbotSvc(chatbotConfigRepo, knowledgeRepo, authRepo, cfg.AIServiceURL, rdb)

	pipelineSvc  := service.NewPipelineStageSvc(pipelineRepo)
	treatmentSvc        := service.NewTreatmentSvc(treatmentRepo, publisher, eventRepo)
	treatmentSessionSvc := service.NewTreatmentSessionSvc(treatmentSessionRepo, treatmentRepo)
	taskSvc             := service.NewTaskSvc(taskRepo)
	ruleSvc      := service.NewRuleSvc(ruleRepo, ruleExecRepo)

	// ── MinIO (storage de branding assets) ──────────────────────────────────
	var brandingSvc domain.BrandingService
	var clinicalNoteSvc domain.ClinicalNoteSvc
	var clinicalFileSvc domain.ClinicalFileSvc
	if cfg.MinIOEndpoint != "" {
		storageCtx, storageCancel := context.WithTimeout(ctx, 10*time.Second)
		storageClient, storageErr := storage.New(storageCtx, storage.Config{
			Endpoint:  cfg.MinIOEndpoint,
			AccessKey: cfg.MinIOAccessKey,
			SecretKey: cfg.MinIOSecretKey,
			UseSSL:    cfg.MinIOUseSSL,
			Bucket:    cfg.MinIOBucket,
			PublicURL: cfg.MinIOPublicURL,
		})
		storageCancel()
		if storageErr != nil {
			slog.Error("MinIO init falló — uploads de branding deshabilitados", "err", storageErr)
		} else {
			brandingSvc = service.NewBrandingSvc(storageClient, authRepo)
			clinicalNoteSvc = service.NewClinicalNoteSvc(clinicalNoteRepo, clinicalFileRepo, apptRepo, storageClient)
			clinicalFileSvc = service.NewClinicalFileSvc(clinicalFileRepo, clinicalNoteRepo, storageClient)
			slog.Info("MinIO listo", "bucket", storageClient.Bucket())
		}
	} else {
		slog.Warn("MINIO_ENDPOINT no configurado — uploads de branding deshabilitados")
	}
	// Clinical History sin MinIO: notas funcionan, uploads deshabilitados
	if clinicalNoteSvc == nil {
		clinicalNoteSvc = service.NewClinicalNoteSvc(clinicalNoteRepo, clinicalFileRepo, apptRepo, nil)
		clinicalFileSvc = service.NewClinicalFileSvc(clinicalFileRepo, clinicalNoteRepo, nil)
	}

	// ── Motor de reglas ──────────────────────────────────────────────────────
	actionRegistry := engine.NewActionRegistry()
	actionRegistry.Register(actions.NewSendWhatsAppAction(waClient))
	actionRegistry.Register(actions.NewCreateTaskAction(taskRepo))
	actionRegistry.Register(actions.NewMoveStageAction(customerRepo, publisher))
	actionRegistry.Register(actions.NewUpdateFieldAction(customerRepo))

	ruleExecutor := engine.NewRuleExecutor(ruleRepo, ruleExecRepo, authRepo, actionRegistry)

	// ── Handlers ──────────────────────────────────────────────────────────────
	authHandler      := handler.NewAuthHandler(authSvc)
	profHandler      := handler.NewProfessionalHandler(profSvc)
	svcHandler       := handler.NewServiceHandler(serviceSvc)
	apptHandler      := handler.NewAppointmentHandler(apptSvc, availSvc)
	pubHandler       := handler.NewPublicHandler(publicSvc)
	waHandler        := handler.NewWhatsAppHandler(waSvc, waClient, cfg.WebhookSecret)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeSvc)
	customerHandler  := handler.NewCustomerHandler(customerRepo)
	settingsHandler  := handler.NewSettingsHandler(authRepo)
	blockHandler     := handler.NewScheduleBlockHandler(scheduleRepo)
	billingHandler   := handler.NewBillingHandler(authRepo, rdb, cfg.StripeSecretKey, cfg.StripeWebhookSecret, cfg.StripePriceStarter, cfg.StripePricePro)
	pipelineHandler  := handler.NewPipelineStageHandler(pipelineSvc)
	treatmentHandler        := handler.NewTreatmentHandler(treatmentSvc)
	treatmentSessionHandler := handler.NewTreatmentSessionHandler(treatmentSessionSvc)
	taskHandler             := handler.NewTaskHandler(taskSvc)
	ruleHandler      := handler.NewRuleHandler(ruleSvc)
	crmHandler       := handler.NewCRMHandler(crmMetricsRepo)
	// brandingSvc puede ser nil si MinIO no está disponible — el handler devuelve 503 en ese caso.
	brandingHandler := handler.NewBrandingHandler(brandingSvc)
	clinicalNoteHandler := handler.NewClinicalNoteHandler(clinicalNoteSvc)
	clinicalFileHandler := handler.NewClinicalFileHandler(clinicalFileSvc)
	chatbotHandler      := handler.NewChatbotHandler(chatbotSvc)

	// ── Workers background ────────────────────────────────────────────────────
	reminderWorker := worker.NewReminderWorker(reminderRepo, notifRepo, waClient)
	outboundWorker := worker.NewOutboundWorker(cfg.RabbitMQURL, waClient, notifRepo)
	go reminderWorker.Start(ctx)
	go outboundWorker.Start(ctx)

	// Worker de mantenimiento de eventos (particiones + limpieza)
	eventsWorker := worker.NewEventsMaintenanceWorker(pool)
	go eventsWorker.Start(ctx)

	// Workers del motor de reglas
	if cfg.RabbitMQURL != "" {
		rulesEventWorker := worker.NewRulesEventWorker(cfg.RabbitMQURL, ruleExecutor)
		go rulesEventWorker.Start(ctx)
	}
	temporalRulesWorker := worker.NewTemporalRulesWorker(ruleRepo, customerRepo, ruleExecutor)
	go temporalRulesWorker.Start(ctx)

	// Re-registrar webhooks de WhatsApp al arrancar.
	// Evolution API puede perder la configuración del webhook al reiniciarse.
	if cfg.WebhookURL != "" {
		go func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			slugs, err := authRepo.FindConnectedTenantSlugs(syncCtx)
			if err != nil {
				slog.Warn("wa.startup: error obteniendo tenants conectados", "error", err)
				return
			}
			endpoint := cfg.WebhookURL + "/api/v1/whatsapp/webhook"
			for _, slug := range slugs {
				if err := waClient.SetWebhook(syncCtx, slug, endpoint); err != nil {
					slog.Warn("wa.startup: webhook no re-registrado", "slug", slug, "error", err)
				} else {
					slog.Info("wa.startup: webhook re-registrado", "slug", slug)
				}
			}
		}()
	}

	// ── Validación de configuración de producción ────────────────────────────
	if cfg.AppEnv == "production" {
		corsOrigins := os.Getenv("CORS_ORIGINS")
		if corsOrigins == "" || corsOrigins == "*" {
			slog.Warn("CORS_ORIGINS no configurado en producción — todos los orígenes permitidos. Configura CORS_ORIGINS con el dominio del frontend.")
		}
	}

	// ── Fiber ─────────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		// El límite debe cubrir archivos clínicos (10 MB) + overhead multipart.
		BodyLimit: 12 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": "Error interno del servidor"})
		},
	})

	// Recover de panics + Request ID en cada request
	app.Use(recover.New())
	app.Use(requestid.New())

	// CORS
	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		if cfg.AppEnv != "production" {
			// Desarrollo: permitir frontend local por defecto
			corsOrigins = "http://localhost:3000,http://127.0.0.1:3000"
		} else {
			corsOrigins = "*"
		}
	}
	// Con credenciales (Authorization, cookies) el navegador no acepta Allow-Origin: *
	allowCreds := corsOrigins != "*"
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowCredentials: allowCreds,
	}))
	if cfg.AppEnv != "production" {
		app.Use(fiberlogger.New())
	}

	// ── Documentación API (Swagger UI + ReDoc) ─────────────────────────────────
	app.Use(swagger.New(swagger.Config{
		BasePath:    "/",
		FileContent: docs.SwaggerJSON,
		Path:        "swagger",
		Title:       "CitaSpot API",
	}))
	app.Get("/openapi.json", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		return c.Send(docs.SwaggerJSON)
	})
	app.Get("/redoc", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(redocHTML)
	})

	// ── Health checks ─────────────────────────────────────────────────────────
	// Liveness: el proceso está arriba
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "citaspot-api"})
	})
	// Readiness: las dependencias críticas están listas
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		checks := fiber.Map{}
		allOk := true

		if err := pool.Ping(c.Context()); err != nil {
			checks["postgres"] = "error: " + err.Error()
			allOk = false
		} else {
			checks["postgres"] = "ok"
		}

		if rdb != nil {
			if err := rdb.Ping(c.Context()).Err(); err != nil {
				checks["redis"] = "error: " + err.Error()
				allOk = false
			} else {
				checks["redis"] = "ok"
			}
		} else {
			checks["redis"] = "unavailable"
		}

		status := "ready"
		statusCode := fiber.StatusOK
		if !allOk {
			status = "degraded"
			statusCode = fiber.StatusServiceUnavailable
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": status, "checks": checks})
	})

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
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)

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
	}), waHandler.Webhook)

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
	}), billingHandler.Webhook)

	// Booking público (sin JWT, por slug)
	pub := api.Group("/public")
	pub.Get("/:slug", pubHandler.GetProfile)
	pub.Get("/:slug/availability", pubHandler.GetAvailability)
	pub.Post("/:slug/book", pubHandler.Book)

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
	protected := api.Group("",
		apiLimiter,
		middleware.JWTMiddleware(cfg.JWTSecret, cfg.SupabaseURL),
		middleware.TenantMiddleware(authRepo, pool),
		middleware.PlanCheckMiddleware(),
	)

	protected.Get("/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"user":   middleware.UserFromContext(c),
			"tenant": middleware.TenantFromContext(c),
		})
	})

	// Marca el onboarding del tenant como completado.
	protected.Post("/onboarding/complete", func(c *fiber.Ctx) error {
		tenantID := middleware.TenantIDFromContext(c)
		if tenantID == uuid.Nil {
			return fiber.NewError(403, "tenant no identificado")
		}
		if err := authRepo.CompleteOnboarding(c.Context(), tenantID); err != nil {
			return fiber.NewError(500, "error interno")
		}

		// Auto-seed CRM por business_type — pipeline aplica a todos los verticales
		// con template; rule templates y knowledge siguen siendo dental-only.
		tenant := middleware.TenantFromContext(c)
		if tenant != nil {
			bt := tenant.BusinessType
			go func() {
				bgCtx := context.Background()
				if seeded, err := seed.SeedPipelineForBusinessType(bgCtx, pool, tenantID, bt); err != nil {
					slog.Warn("onboarding: error seeding pipeline", "tenant_id", tenantID, "business_type", bt, "error", err)
				} else if len(seeded) > 0 {
					slog.Info("onboarding: pipeline seeded", "tenant_id", tenantID, "business_type", bt, "stages", len(seeded))
				}
				// Templates genericos vertical-agnosticos (post-cita, etc) — corren para todos los tenants.
				if err := seed.SeedGenericRuleTemplates(bgCtx, pool); err != nil {
					slog.Warn("onboarding: error seeding generic rule templates", "error", err)
				}
				// Copiar reglas de notificación de citas al tenant nuevo
				if err := seed.SeedAppointmentRulesForTenant(bgCtx, pool, tenantID); err != nil {
					slog.Warn("onboarding: error seeding appointment rules", "tenant_id", tenantID, "error", err)
				} else {
					slog.Info("onboarding: appointment notification rules seeded", "tenant_id", tenantID)
				}
				if bt == "dental" {
					// Rule templates son globales (idempotent) — safe to call multiple times
					if err := seed.SeedDentalRuleTemplates(bgCtx, pool); err != nil {
						slog.Warn("onboarding: error seeding dental rule templates", "error", err)
					}
					if err := seed.SeedDentalKnowledge(bgCtx, pool, tenantID); err != nil {
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
	profs.Get("/", profHandler.List)
	profs.Post("/", profHandler.Create)
	profs.Get("/:id", profHandler.GetByID)
	profs.Patch("/:id", profHandler.Update)
	profs.Get("/:id/schedule", profHandler.GetSchedule)
	profs.Put("/:id/schedule", profHandler.SetSchedule)
	profs.Get("/:id/services", profHandler.ListServices)
	profs.Post("/:id/services/:serviceID", profHandler.AssignService)
	profs.Delete("/:id/services/:serviceID", profHandler.RemoveService)

	srvs := protected.Group("/services")
	srvs.Get("/", svcHandler.List)
	srvs.Post("/", svcHandler.Create)
	srvs.Get("/:id", svcHandler.GetByID)
	srvs.Patch("/:id", svcHandler.Update)
	srvs.Delete("/:id", svcHandler.Delete)

	appts := protected.Group("/appointments")
	appts.Get("/availability", apptHandler.Availability)
	appts.Get("/search", apptHandler.ListFiltered)
	appts.Get("/", apptHandler.List)
	appts.Post("/", apptHandler.Create)
	appts.Get("/:id", apptHandler.GetByID)
	appts.Patch("/:id", apptHandler.Update)
	appts.Delete("/:id/cancel", apptHandler.Cancel)
	appts.Patch("/:id/reschedule", apptHandler.Reschedule)

	knowledge := protected.Group("/knowledge")
	knowledge.Get("/", knowledgeHandler.List)
	knowledge.Post("/", knowledgeHandler.Create)
	knowledge.Post("/upload", knowledgeHandler.Upload)
	knowledge.Get("/:id", knowledgeHandler.Get)
	knowledge.Put("/:id", knowledgeHandler.Update)
	knowledge.Delete("/:id", knowledgeHandler.Delete)

	chatbot := protected.Group("/chatbot")
	chatbot.Get("/config", chatbotHandler.GetConfig)
	chatbot.Patch("/config", chatbotHandler.UpdateConfig)
	chatbot.Post("/test", chatbotHandler.Test)
	chatbot.Post("/validate", chatbotHandler.Validate)

	customers := protected.Group("/customers")
	customers.Get("/", customerHandler.List)
	customers.Get("/:id", customerHandler.GetByID)
	customers.Patch("/:id/stage", customerHandler.UpdateStage)

	protected.Get("/whatsapp/status", waHandler.Status)
	protected.Get("/whatsapp/qr", waHandler.GetQR)
	protected.Post("/whatsapp/connect", waHandler.Connect)
	protected.Delete("/whatsapp/disconnect", waHandler.Disconnect)

	// Billing: checkout, subscription info, invoices y cancel requieren auth
	billing := protected.Group("/billing")
	billing.Post("/checkout", billingHandler.CreateCheckout)
	billing.Get("/subscription", billingHandler.Subscription)
	billing.Get("/invoices", billingHandler.Invoices)
	billing.Post("/cancel", billingHandler.CancelSubscription)

	// Settings
	protected.Get("/settings", settingsHandler.Get)
	protected.Patch("/settings", settingsHandler.Update)
	protected.Patch("/tenant/profile", settingsHandler.UpdateTenantProfile)
	protected.Patch("/me/profile", settingsHandler.UpdateMyProfile)

	// Tenant branding (logo + portada). Las rutas siempre se registran.
	// Si MinIO no está disponible, el handler devuelve 503 con mensaje descriptivo.
	branding := protected.Group("/tenant/branding")
	branding.Post("/logo", brandingHandler.UploadLogo)
	branding.Post("/cover", brandingHandler.UploadCover)
	branding.Delete("/:kind", brandingHandler.Remove)

	// Schedule Blocks
	blocks := protected.Group("/schedule-blocks")
	blocks.Post("/", blockHandler.Create)
	blocks.Get("/", blockHandler.List)
	blocks.Delete("/:id", blockHandler.Delete)

	// Pipeline Stages
	stages := protected.Group("/pipeline-stages")
	stages.Get("/", pipelineHandler.List)
	stages.Post("/", pipelineHandler.Create)
	stages.Post("/load-template", func(c *fiber.Ctx) error {
		tenantID := middleware.TenantIDFromContext(c)
		if tenantID == uuid.Nil {
			return fiber.NewError(403, "tenant no identificado")
		}
		tenant := middleware.TenantFromContext(c)
		if tenant == nil {
			return fiber.NewError(403, "tenant no identificado")
		}
		seeded, err := seed.SeedPipelineForBusinessType(c.Context(), pool, tenantID, tenant.BusinessType)
		if err != nil {
			slog.Error("pipeline load-template: failed", "tenant_id", tenantID, "error", err)
			return fiber.NewError(500, "error interno")
		}
		if len(seeded) == 0 {
			return fiber.NewError(409, "el pipeline ya tiene etapas o el vertical no tiene template")
		}
		return c.JSON(fiber.Map{"ok": true, "stages": seeded})
	})
	stages.Put("/reorder", pipelineHandler.Reorder)
	stages.Get("/:id", pipelineHandler.GetByID)
	stages.Patch("/:id", pipelineHandler.Update)
	stages.Delete("/:id", pipelineHandler.Delete)

	// Treatments
	treatments := protected.Group("/treatments")
	treatments.Get("/", treatmentHandler.List)
	treatments.Post("/", treatmentHandler.Create)
	treatments.Get("/:id", treatmentHandler.GetByID)
	treatments.Patch("/:id", treatmentHandler.Update)
	treatments.Patch("/:id/status", treatmentHandler.UpdateStatus)

	// Sesiones de tratamiento
	treatmentSessions := treatments.Group("/:id/sessions")
	treatmentSessions.Get("/", treatmentSessionHandler.List)
	treatmentSessions.Post("/", treatmentSessionHandler.Create)
	treatmentSessions.Get("/:sid", treatmentSessionHandler.GetByID)
	treatmentSessions.Patch("/:sid", treatmentSessionHandler.Update)
	treatmentSessions.Delete("/:sid", treatmentSessionHandler.Delete)

	// Tasks
	tasks := protected.Group("/tasks")
	tasks.Get("/", taskHandler.List)
	tasks.Post("/", taskHandler.Create)
	tasks.Get("/:id", taskHandler.GetByID)
	tasks.Patch("/:id", taskHandler.Update)
	tasks.Post("/:id/complete", taskHandler.Complete)
	tasks.Post("/:id/dismiss", taskHandler.Dismiss)

	// Rules
	rules := protected.Group("/rules")
	rules.Get("/", ruleHandler.List)
	rules.Post("/", ruleHandler.Create)
	rules.Get("/:id", ruleHandler.GetByID)
	rules.Patch("/:id", ruleHandler.Update)
	rules.Delete("/:id", ruleHandler.Delete)
	rules.Get("/:id/executions", ruleHandler.ListExecutions)

	// Clinical Notes
	appts.Post("/:id/clinical-note", clinicalNoteHandler.Create)
	appts.Get("/:id/clinical-note", clinicalNoteHandler.GetByAppointment)
	customers.Get("/:id/clinical-notes", clinicalNoteHandler.ListByCustomer)
	customers.Get("/:id/clinical-notes/:noteId", clinicalNoteHandler.GetByID)
	clinicalNotes := protected.Group("/clinical-notes")
	clinicalNotes.Patch("/:noteId", clinicalNoteHandler.Update)
	clinicalNotes.Delete("/:noteId", clinicalNoteHandler.Delete)
	clinicalNotes.Post("/:noteId/files/upload", clinicalFileHandler.Upload)
	clinicalNotes.Get("/:noteId/files", clinicalFileHandler.ListByNote)

	// Clinical Files
	customers.Get("/:id/clinical-files", clinicalFileHandler.ListByCustomer)
	protected.Delete("/clinical-files/:fileId", clinicalFileHandler.Delete)

	// CRM Metrics
	protected.Get("/crm/metrics", crmHandler.Metrics)

	// ── Arrancar servidor ─────────────────────────────────────────────────────
	slog.Info("Core API iniciando", "port", cfg.Port, "env", cfg.AppEnv)
	serverErr := make(chan error, 1)
	go func() { serverErr <- app.Listen(":" + cfg.Port) }()

	select {
	case <-ctx.Done():
		slog.Info("Core API: apagando...")
		_ = app.Shutdown()
	case err := <-serverErr:
		logger.Fatal("Error iniciando servidor", "error", err)
	}
}
