package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
)

// ── JWKS — claves públicas de Supabase para ES256 ─────────────────────────────

type jwkEntry struct {
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type jwksResponse struct {
	Keys []jwkEntry `json:"keys"`
}

// loadJWKS descarga las claves públicas del endpoint JWKS de Supabase.
// Las claves se cachean en memoria — solo se cargan una vez al arrancar.
func loadJWKS(supabaseURL string) map[string]*ecdsa.PublicKey {
	keys := make(map[string]*ecdsa.PublicKey)
	if supabaseURL == "" {
		return keys
	}

	url := strings.TrimRight(supabaseURL, "/") + "/auth/v1/.well-known/jwks.json"
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		slog.Warn("[JWT] no se pudo obtener JWKS", "url", url, "error", err)
		return keys
	}
	defer resp.Body.Close()

	var data jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		slog.Warn("[JWT] error parseando JWKS", "error", err)
		return keys
	}

	for _, k := range data.Keys {
		if k.Kty != "EC" || k.Alg != "ES256" || k.Crv != "P-256" {
			continue
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			continue
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			continue
		}
		pub := &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}
		keys[k.Kid] = pub
		slog.Info("[JWT] clave EC cargada", "kid", k.Kid)
	}

	return keys
}

// ── JWTMiddleware ──────────────────────────────────────────────────────────────

// JWTMiddleware valida el Bearer token emitido por Supabase Auth.
// Soporta HS256 (tokens legacy/anon) y ES256 (tokens de usuario actuales).
// Extrae el claim "sub" y lo almacena en el contexto para TenantMiddleware.
func JWTMiddleware(jwtSecret, supabaseURL string) fiber.Handler {
	secretBytes := []byte(jwtSecret)
	ecKeys := loadJWKS(supabaseURL)

	return func(c *fiber.Ctx) error {
		// Extraer token del header Authorization: Bearer <token>
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token de acceso requerido",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Formato de autorización inválido",
			})
		}
		tokenStr := parts[1]

		// Parsear y validar — soporta HS256 y ES256 (tokens Supabase modernos)
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			switch t.Method.(type) {
			case *jwt.SigningMethodECDSA:
				// ES256 — verificar con clave pública del JWKS de Supabase
				kid, _ := t.Header["kid"].(string)
				if kid == "" {
					return nil, fmt.Errorf("token ES256 sin campo kid")
				}
				pub, ok := ecKeys[kid]
				if !ok {
					return nil, fmt.Errorf("kid %q no encontrado en JWKS", kid)
				}
				return pub, nil

			case *jwt.SigningMethodHMAC:
				// HS256 — verificar con el secret compartido (JWT_SECRET)
				return secretBytes, nil

			default:
				return nil, jwt.ErrSignatureInvalid
			}
		}, jwt.WithValidMethods([]string{"HS256", "ES256"}))

		if err != nil || !token.Valid {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido o expirado",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token inválido",
			})
		}

		// "sub" en Supabase Auth JWTs = UUID del usuario en auth.users
		sub, _ := claims["sub"].(string)
		if sub == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token sin identidad",
			})
		}

		// Guardar auth_id en contexto para que TenantMiddleware lo use
		c.Locals(ctxKeyAuthID, sub)

		return c.Next()
	}
}
