// Entrypoint del Core API de CitaSpot.
// Solo configuración y arranque — la lógica va en los paquetes internos.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // Embebe la base de datos de timezones en el binario (Alpine/scratch no la incluyen)

	"github.com/gofiber/contrib/swagger"
	"github.com/joho/godotenv"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
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
	"github.com/citaspot/api/internal/repository"
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
	userNotifRepo := repository.NewUserNotificationRepository(pool)
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
	waitlistRepo      := repository.NewWaitlistRepository(pool)
	tenantModuleRepo  := repository.NewTenantModuleRepository(pool)

	// ── Servicios ─────────────────────────────────────────────────────────────
	authSvc    := service.NewAuthService(authRepo, cfg)
	profSvc    := service.NewProfessionalService(profRepo, scheduleRepo, serviceRepo, apptRepo, publisher)
	serviceSvc := service.NewServiceSvc(serviceRepo, publisher)
	availSvc   := service.NewAvailabilityService(scheduleRepo, serviceRepo)
	apptSvc    := service.NewAppointmentSvc(apptRepo, serviceRepo, customerRepo, profRepo, authRepo, waClient, notifRepo, publisher, eventRepo, rdb)
	publicSvc  := service.NewPublicSvc(authRepo, profRepo, serviceRepo, availSvc, apptSvc, customerRepo, treatmentRepo, tenantModuleRepo)

	// Feed in-app del dashboard (CITAS-41). Publica al canal Redis
	// `tenant:{id}:notifications` que el handler SSE multiplexa con appointments.
	userNotifSvc := service.NewUserNotificationSvc(userNotifRepo, rdb)
	// Inyectar via setter para no romper el constructor de appointmentSvc.
	if setter, ok := apptSvc.(interface {
		SetUserNotificationSvc(domain.UserNotificationSvc)
	}); ok {
		setter.SetUserNotificationSvc(userNotifSvc)
	}
	// Mismo patrón para publicSvc — usa notifSvc en NotifyHandoff (guards).
	if setter, ok := publicSvc.(interface {
		SetUserNotificationSvc(domain.UserNotificationSvc)
	}); ok {
		setter.SetUserNotificationSvc(userNotifSvc)
	}

	var waSvc domain.WhatsAppSvc
	if publisher != nil {
		waSvc = service.NewWhatsAppSvc(authRepo, convRepo, customerRepo, publisher, eventRepo)
		if setter, ok := waSvc.(interface {
			SetUserNotificationSvc(domain.UserNotificationSvc)
		}); ok {
			setter.SetUserNotificationSvc(userNotifSvc)
		}
	}

	// publisher puede ser nil si RabbitMQ no está disponible (modo degradado)
	knowledgeSvc := service.NewKnowledgeSvc(knowledgeRepo, publisher)
	chatbotSvc   := service.NewChatbotSvc(chatbotConfigRepo, knowledgeRepo, authRepo, cfg.AIServiceURL, rdb)

	pipelineSvc  := service.NewPipelineStageSvc(pipelineRepo)
	treatmentSvc        := service.NewTreatmentSvc(treatmentRepo, publisher, eventRepo)
	treatmentSessionSvc := service.NewTreatmentSessionSvc(treatmentSessionRepo, treatmentRepo)
	taskSvc             := service.NewTaskSvc(taskRepo)
	ruleSvc      := service.NewRuleSvc(ruleRepo, ruleExecRepo)
	waitlistSvc  := service.NewWaitlistSvc(waitlistRepo)

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
	customerHandler  := handler.NewCustomerHandler(customerRepo, publisher)
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
	waitlistHandler     := handler.NewWaitlistHandler(waitlistSvc)
	// Realtime (SSE) — Phase C: suscripción por conexión a Redis Pub/Sub.
	// Si rdb es nil (Redis no disponible al startup), el handler responde 503.
	realtimeHandler     := handler.NewRealtimeHandler(rdb)
	// Notificaciones in-app del dashboard (CITAS-41)
	userNotifHandler    := handler.NewUserNotificationHandler(userNotifSvc)

	// ── Workers background ────────────────────────────────────────────────────
	reminderWorker := worker.NewReminderWorker(reminderRepo, notifRepo, waClient, publisher)
	outboundWorker := worker.NewOutboundWorker(cfg.RabbitMQURL, waClient, notifRepo, convRepo)
	dentalNotificationsWorker := worker.NewDentalNotificationsWorker(treatmentRepo, treatmentSessionRepo, notifRepo, waClient)
	go reminderWorker.Start(ctx)
	go outboundWorker.Start(ctx)
	go dentalNotificationsWorker.Start(ctx)

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
		// El access logger por defecto incluye el query string en la URL — eso
		// expondría el JWT pasado como ?token=... en los endpoints SSE. Saltamos
		// /realtime/* aquí; el grupo realtime tiene su propio scrubber que loguea
		// SOLO method/path/ip/status sin query string. Ver cmd/server/routes.go.
		app.Use(fiberlogger.New(fiberlogger.Config{
			Next: func(c *fiber.Ctx) bool {
				return strings.HasPrefix(c.Path(), "/api/v1/realtime/")
			},
		}))
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

	// ── Wiring de rutas de negocio ────────────────────────────────────────────
	// Extraído a routes.go para permitir tests del wiring (ver routes_test.go).
	SetupRoutes(app, &RouteDeps{
		Cfg:                     cfg,
		Pool:                    pool,
		Rdb:                     rdb,
		AuthRepo:                authRepo,
		TenantModuleRepo:        tenantModuleRepo,
		ProfSvc:                 profSvc,
		AuthHandler:             authHandler,
		ProfHandler:             profHandler,
		SvcHandler:              svcHandler,
		ApptHandler:             apptHandler,
		PubHandler:              pubHandler,
		WaHandler:               waHandler,
		KnowledgeHandler:        knowledgeHandler,
		CustomerHandler:         customerHandler,
		SettingsHandler:         settingsHandler,
		BlockHandler:            blockHandler,
		BillingHandler:          billingHandler,
		PipelineHandler:         pipelineHandler,
		TreatmentHandler:        treatmentHandler,
		TreatmentSessionHandler: treatmentSessionHandler,
		TaskHandler:             taskHandler,
		RuleHandler:             ruleHandler,
		CrmHandler:              crmHandler,
		BrandingHandler:         brandingHandler,
		ClinicalNoteHandler:     clinicalNoteHandler,
		ClinicalFileHandler:     clinicalFileHandler,
		ChatbotHandler:          chatbotHandler,
		WaitlistHandler:         waitlistHandler,
		RealtimeHandler:         realtimeHandler,
		UserNotifHandler:        userNotifHandler,
	})

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
