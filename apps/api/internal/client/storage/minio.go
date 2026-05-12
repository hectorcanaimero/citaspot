// Package storage provee acceso a MinIO (S3-compat) para subir y servir assets de tenants.
package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client envuelve minio-go con la config del proyecto.
type Client struct {
	mc        *minio.Client
	bucket    string
	publicURL string
}

// Config para inicializar el cliente.
type Config struct {
	Endpoint  string // host:port sin scheme
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
	PublicURL string // base URL pública para construir las URLs servidas al browser
}

// New crea el cliente y verifica que el bucket exista (lo crea con policy pública si no).
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("storage.New: MINIO_ENDPOINT no configurado")
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage.New: %w", err)
	}

	c := &Client{
		mc:        mc,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
	}
	if err := c.ensureBucket(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// ensureBucket crea el bucket si no existe.
// SetBucketPolicy es best-effort: muchos proveedores externos (R2, S3, etc.)
// no permiten modificar policies vía API — el acceso público se configura en el panel.
func (c *Client) ensureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("storage.ensureBucket: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("storage.MakeBucket: %w", err)
		}
	}
	// Intenta aplicar policy de lectura pública. Si el proveedor no lo soporta,
	// se ignora — el acceso público debe configurarse manualmente en el panel del proveedor.
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"AWS": ["*"]},
			"Action": ["s3:GetObject"],
			"Resource": ["arn:aws:s3:::%s/*"]
		}]
	}`, c.bucket)
	if err := c.mc.SetBucketPolicy(ctx, c.bucket, policy); err != nil {
		slog.Warn("storage.SetBucketPolicy: ignorado (configurá acceso público manualmente en tu proveedor)", "bucket", c.bucket, "err", err)
	}
	return nil
}

// Put sube un objeto y devuelve la URL pública.
func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := c.mc.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return "", fmt.Errorf("storage.Put: %w", err)
	}
	return c.publicURLFor(key), nil
}

// Delete elimina un objeto. No falla si no existe.
func (c *Client) Delete(ctx context.Context, key string) error {
	err := c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage.Delete: %w", err)
	}
	return nil
}

// KeyFromURL extrae la key del bucket a partir de una URL pública generada por Put.
// Devuelve "" si la URL no pertenece a este bucket.
func (c *Client) KeyFromURL(url string) string {
	prefix := c.publicURLFor("")
	if !strings.HasPrefix(url, prefix) {
		return ""
	}
	return strings.TrimPrefix(url, prefix)
}

// Bucket retorna el nombre del bucket configurado.
func (c *Client) Bucket() string { return c.bucket }

func (c *Client) publicURLFor(key string) string {
	return fmt.Sprintf("%s/%s/%s", c.publicURL, c.bucket, key)
}
