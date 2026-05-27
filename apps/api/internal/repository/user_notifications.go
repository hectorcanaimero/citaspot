package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/citaspot/api/internal/domain"
)

type userNotificationRepository struct {
	db *pgxpool.Pool
}

// NewUserNotificationRepository crea el repositorio de notificaciones in-app.
func NewUserNotificationRepository(db *pgxpool.Pool) domain.UserNotificationRepository {
	return &userNotificationRepository{db: db}
}

// Create inserta una nueva notificación. Si el caller no setea ID o CreatedAt,
// se asignan aquí.
func (r *userNotificationRepository) Create(ctx context.Context, n *domain.UserNotification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	// metadata por defecto: objeto vacío
	if len(n.Metadata) == 0 {
		n.Metadata = json.RawMessage(`{}`)
	}

	return withTenant(ctx, r.db, n.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_notifications (
				id, tenant_id, type, title, body, metadata, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, n.ID, n.TenantID, n.Type, n.Title, nullableString(n.Body), []byte(n.Metadata), n.CreatedAt)
		if err != nil {
			return fmt.Errorf("userNotificationRepository.Create: %w", err)
		}
		return nil
	})
}

// ListByTenant pagina con cursor sobre created_at DESC.
// Estrategia: pedimos limit+1 filas y, si volvieron limit+1, descartamos la
// última y devolvemos su created_at como nextCursor.
func (r *userNotificationRepository) ListByTenant(
	ctx context.Context, tenantID uuid.UUID, limit int, cursor *time.Time,
) ([]*domain.UserNotification, *time.Time, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	result := make([]*domain.UserNotification, 0, limit)
	var nextCursor *time.Time

	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var rows pgx.Rows
		var err error
		if cursor != nil {
			rows, err = tx.Query(ctx, `
				SELECT id, tenant_id, type, title, body, metadata, read_at, created_at
				FROM user_notifications
				WHERE tenant_id = $1
				  AND created_at < $2
				ORDER BY created_at DESC
				LIMIT $3
			`, tenantID, *cursor, limit+1)
		} else {
			rows, err = tx.Query(ctx, `
				SELECT id, tenant_id, type, title, body, metadata, read_at, created_at
				FROM user_notifications
				WHERE tenant_id = $1
				ORDER BY created_at DESC
				LIMIT $2
			`, tenantID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("userNotificationRepository.ListByTenant: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			n := &domain.UserNotification{}
			var body *string
			var meta []byte
			if err := rows.Scan(
				&n.ID, &n.TenantID, &n.Type, &n.Title, &body, &meta, &n.ReadAt, &n.CreatedAt,
			); err != nil {
				return fmt.Errorf("userNotificationRepository.ListByTenant: scan: %w", err)
			}
			if body != nil {
				n.Body = *body
			}
			if len(meta) == 0 {
				n.Metadata = json.RawMessage(`{}`)
			} else {
				n.Metadata = json.RawMessage(meta)
			}
			result = append(result, n)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, nil, err
	}

	// Si hay una fila extra, hubo página siguiente. Devolvemos su created_at
	// como cursor y la quitamos del slice.
	if len(result) > limit {
		extra := result[limit]
		nextCursor = &extra.CreatedAt
		result = result[:limit]
	}
	return result, nextCursor, nil
}

// UnreadCount cuenta filas con read_at IS NULL — usa el partial index.
func (r *userNotificationRepository) UnreadCount(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM user_notifications
			WHERE tenant_id = $1
			  AND read_at IS NULL
		`, tenantID).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("userNotificationRepository.UnreadCount: %w", err)
	}
	return count, nil
}

// MarkRead marca una notificación como leída. Si la fila no existe o ya
// estaba leída, retorna ErrUserNotificationNotFound.
func (r *userNotificationRepository) MarkRead(ctx context.Context, tenantID, id uuid.UUID) error {
	return withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE user_notifications
			SET read_at = NOW()
			WHERE tenant_id = $1
			  AND id        = $2
			  AND read_at IS NULL
		`, tenantID, id)
		if err != nil {
			return fmt.Errorf("userNotificationRepository.MarkRead: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrUserNotificationNotFound
		}
		return nil
	})
}

// MarkAllRead marca todas las no-leídas como leídas. Retorna cantidad afectada.
func (r *userNotificationRepository) MarkAllRead(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var affected int
	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE user_notifications
			SET read_at = NOW()
			WHERE tenant_id = $1
			  AND read_at IS NULL
		`, tenantID)
		if err != nil {
			return fmt.Errorf("userNotificationRepository.MarkAllRead: %w", err)
		}
		affected = int(tag.RowsAffected())
		return nil
	})
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// nullableString convierte string vacío a *string nil, para usar con NULL en DB.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
