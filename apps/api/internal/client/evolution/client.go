// Package evolution contiene el cliente HTTP para la Evolution API (WhatsApp).
package evolution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/citaspot/api/internal/domain"
)

// Client implementa domain.WAClient para interactuar con Evolution API.
type Client struct {
	baseURL    string
	apiKey     string
	webhookURL string // URL pública para recibir eventos; vacío en desarrollo
	http       *http.Client
}

// New crea un cliente de Evolution API.
// webhookURL es la URL pública de este servidor (ej: "https://api.tudominio.com").
// Si está vacío, Connect no configurará el webhook automáticamente.
func New(baseURL, apiKey, webhookURL string) domain.WAClient {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		webhookURL: webhookURL,
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

// SendText envía un mensaje de texto vía Evolution API.
// Retorna el wa_message_id del mensaje enviado.
func (c *Client) SendText(ctx context.Context, instanceName, phone, text string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"number": phone,
		"text":   text,
	})

	url := fmt.Sprintf("%s/message/sendText/%s", c.baseURL, instanceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("evolution.SendText: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("evolution.SendText: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// Error de cliente (número no existe, etc.) — fallo permanente, no reintentar.
		return "", fmt.Errorf("evolution.SendText: status %d: %s: %w", resp.StatusCode, string(respBody), domain.ErrWAPermanentFailure)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("evolution.SendText: status %d: %s", resp.StatusCode, string(respBody))
	}

	// Extraer message ID de la respuesta
	var result struct {
		Key struct {
			ID string `json:"id"`
		} `json:"key"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", nil // no fatal — el mensaje fue enviado
	}
	return result.Key.ID, nil
}

// Connect crea la instancia en Evolution API (si no existe) e inicia la conexión.
// Si webhookURL está configurado, registra automáticamente el webhook por instancia.
// Retorna el QR base64 si Evolution lo incluye en la respuesta de /instance/connect.
func (c *Client) Connect(ctx context.Context, instanceName string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"instanceName": instanceName,
		"qrcode":       true,
		"integration":  "WHATSAPP-BAILEYS",
	})

	// 1. Crear instancia (si ya existe, Evolution devuelve 4xx — lo ignoramos)
	createURL := fmt.Sprintf("%s/instance/create", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("evolution.Connect: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("evolution.Connect: http: %w", err)
	}
	resp.Body.Close()

	// 2. Configurar webhook (best-effort)
	if c.webhookURL != "" {
		webhookEndpoint := c.webhookURL + "/api/v1/whatsapp/webhook"
		if err := c.SetWebhook(ctx, instanceName, webhookEndpoint); err != nil {
			fmt.Printf("evolution.Connect: SetWebhook warning: %v\n", err)
		}
	}

	// 3. Iniciar conexión — fuerza la generación del QR aunque la instancia ya exista
	connectURL := fmt.Sprintf("%s/instance/connect/%s", c.baseURL, instanceName)
	reqConnect, err := http.NewRequestWithContext(ctx, http.MethodGet, connectURL, nil)
	if err != nil {
		return "", fmt.Errorf("evolution.Connect: build connect request: %w", err)
	}
	reqConnect.Header.Set("apikey", c.apiKey)

	respConnect, err := c.http.Do(reqConnect)
	if err != nil {
		return "", fmt.Errorf("evolution.Connect: connect http: %w", err)
	}
	defer respConnect.Body.Close()

	// Capturar el QR de la respuesta (si viene)
	if respConnect.StatusCode != http.StatusOK {
		return "", nil
	}

	respBody, err := io.ReadAll(respConnect.Body)
	if err != nil {
		return "", nil
	}

	// Parsear base64 de la respuesta de Evolution
	var result struct {
		Base64 string `json:"base64"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Base64 != "" {
		return result.Base64, nil
	}

	return "", nil
}

// SetWebhook configura la URL de webhook para una instancia específica en Evolution API.
func (c *Client) SetWebhook(ctx context.Context, instanceName, webhookURL string) error {
	body, _ := json.Marshal(map[string]any{
		"webhook": map[string]any{
			"enabled":         true,
			"url":             webhookURL,
			"webhookByEvents": false,
			"webhookBase64":   true,
			"events": []string{
				"MESSAGES_UPSERT",
				"CONNECTION_UPDATE",
				"QRCODE_UPDATED",
			},
		},
	})

	url := fmt.Sprintf("%s/webhook/set/%s", c.baseURL, instanceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("evolution.SetWebhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("evolution.SetWebhook: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("evolution.SetWebhook: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// FetchQR obtiene el QR code actual directamente de Evolution API.
// Útil cuando el webhook QRCODE_UPDATED no llega (ej: desarrollo sin URL pública).
// Retorna string vacío si la instancia no tiene QR disponible.
func (c *Client) FetchQR(ctx context.Context, instanceName string) (string, error) {
	url := fmt.Sprintf("%s/instance/connect/%s", c.baseURL, instanceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("evolution.FetchQR: build request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("evolution.FetchQR: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil // instancia no existe o ya conectada — no es error fatal
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	// Evolution v2 puede devolver el base64 en la raíz o dentro de un objeto anidado.
	// Intentar primero la estructura plana: {"base64": "data:image/png;base64,..."}
	var flat struct {
		Base64 string `json:"base64"`
	}
	if err := json.Unmarshal(body, &flat); err == nil && flat.Base64 != "" {
		return flat.Base64, nil
	}

	// Fallback: estructura anidada de Evolution v2.2+
	// {"qrcode": {"base64": "..."}} o {"instance": {...}, "qrcode": {"base64": "..."}}
	var nested struct {
		QRCode struct {
			Base64 string `json:"base64"`
		} `json:"qrcode"`
	}
	if err := json.Unmarshal(body, &nested); err == nil && nested.QRCode.Base64 != "" {
		return nested.QRCode.Base64, nil
	}

	return "", nil
}

// Disconnect cierra la sesión de WhatsApp del tenant.
func (c *Client) Disconnect(ctx context.Context, instanceName string) error {
	url := fmt.Sprintf("%s/instance/logout/%s", c.baseURL, instanceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("evolution.Disconnect: build request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("evolution.Disconnect: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("evolution.Disconnect: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// IsConnected verifica si la sesión de WhatsApp está activa.
func (c *Client) IsConnected(ctx context.Context, instanceName string) (bool, error) {
	url := fmt.Sprintf("%s/instance/connectionState/%s", c.baseURL, instanceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("evolution.IsConnected: build request: %w", err)
	}
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("evolution.IsConnected: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Instance struct {
			State string `json:"state"`
		} `json:"instance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil
	}
	return result.Instance.State == "open", nil
}
