// Package config centraliza la configuración del Core API desde variables de entorno.
package config

import (
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// Carga .env si existe (no falla si no está — en producción se usan env vars directas)
	godotenv.Load()
}

// Config contiene toda la configuración de la aplicación.
type Config struct {
	Port    string
	AppEnv  string
	LogLevel string

	// Base de datos
	DatabaseURL string

	// Cache
	RedisURL string

	// Mensajería
	RabbitMQURL string

	// Supabase Auth
	JWTSecret          string // JWT Secret del proyecto Supabase (Settings > API > JWT Settings)
	SupabaseURL        string // Project URL (ej: https://xxx.supabase.co)
	SupabaseAnonKey    string // Anon key — para sign-in desde servidor
	SupabaseServiceKey string // Service role key — para Admin API (crear usuarios)

	// WhatsApp (Evolution API)
	EvolutionAPIURL string
	EvolutionAPIKey string
	WebhookURL      string // URL pública de este servidor para recibir eventos de Evolution
	WebhookSecret   string // Secret para validar llamadas entrantes del webhook de Evolution (opcional en dev)

	// AI Service
	AIServiceURL string

	// Pagos (Stripe)
	StripeSecretKey     string
	StripeWebhookSecret string
	StripePriceStarter  string // Price ID del plan Starter ($10/mes)
	StripePricePro      string // Price ID del plan Professional ($25/mes)
}

// Load carga la configuración desde variables de entorno con defaults para desarrollo.
func Load() *Config {
	return &Config{
		Port:     getEnv("PORT", "3001"),
		AppEnv:   getEnv("APP_ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "debug"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:citaspot123@168.231.94.201:5432/citaspot"),
		RedisURL:    getEnv("REDIS_URL", "redis://default:citaspot123@168.231.94.201:9876/0"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://citaspot:citaspot123@187.127.1.191:32777/"),

		JWTSecret:          getEnv("JWT_SECRET", ""),
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:    getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseServiceKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),

		EvolutionAPIURL: getEnv("EVOLUTION_API_URL", "http://localhost:8080"),
		EvolutionAPIKey: getEnv("EVOLUTION_API_KEY", ""),
		WebhookURL:      getEnv("WEBHOOK_URL", ""),
		WebhookSecret:   getEnv("WEBHOOK_SECRET", ""),

		AIServiceURL: getEnv("AI_SERVICE_URL", "http://localhost:8001"),

		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePriceStarter:  getEnv("STRIPE_PRICE_STARTER", ""),
		StripePricePro:      getEnv("STRIPE_PRICE_PRO", ""),
	}
}

// IsDevelopment retorna true si el entorno es desarrollo.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
