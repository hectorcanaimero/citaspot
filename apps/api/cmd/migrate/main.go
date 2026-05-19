// Runner de migraciones para CitaSpot.
// Aplica archivos SQL en orden desde db/migrations/ y registra en schema_migrations.
// Uso: go run cmd/migrate/main.go [up|down N|status]
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// migrationsDir es la ruta a los archivos SQL desde la raíz de apps/api/
const migrationsDir = "./db/migrations"

func main() {
	command := "up"
	count := 1

	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if len(os.Args) > 2 {
		count, _ = strconv.Atoi(os.Args[2])
	}

	// Conexión directa al host para desarrollo (Docker expone :5432)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://citaspot:citaspot_dev@localhost:5432/citaspot"
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		slog.Error("No se pudo conectar a la base de datos", "error", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// Crear tabla de seguimiento si no existe
	if err := ensureMigrationsTable(ctx, conn); err != nil {
		slog.Error("Error creando tabla schema_migrations", "error", err)
		os.Exit(1)
	}

	switch command {
	case "up":
		if err := runUp(ctx, conn); err != nil {
			slog.Error("Error aplicando migraciones", "error", err)
			os.Exit(1)
		}
	case "down":
		if err := runDown(ctx, conn, count); err != nil {
			slog.Error("Error revirtiendo migraciones", "error", err)
			os.Exit(1)
		}
	case "status":
		if err := showStatus(ctx, conn); err != nil {
			slog.Error("Error obteniendo estado", "error", err)
			os.Exit(1)
		}
	default:
		slog.Error("Comando desconocido", "command", command, "usage", "up | down [N] | status")
		os.Exit(1)
	}
}

// ensureMigrationsTable crea la tabla de tracking si no existe.
func ensureMigrationsTable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     VARCHAR(255) PRIMARY KEY,
			applied_at  TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	return err
}

// getMigrationFiles retorna los archivos .sql del directorio de migraciones, ordenados.
func getMigrationFiles() ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("leyendo directorio %s: %w", migrationsDir, err)
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		// Excluir archivos .down.sql: son scripts de reversión, no migraciones forward
		if strings.HasSuffix(name, ".down.sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files, nil
}

// getApplied retorna un set de versiones ya aplicadas.
func getApplied(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

// runUp aplica todas las migraciones pendientes en orden, cada una en su propia transacción.
func runUp(ctx context.Context, conn *pgx.Conn) error {
	files, err := getMigrationFiles()
	if err != nil {
		return err
	}

	applied, err := getApplied(ctx, conn)
	if err != nil {
		return fmt.Errorf("obteniendo migraciones aplicadas: %w", err)
	}

	pendingCount := 0
	for _, file := range files {
		if applied[file] {
			slog.Info("omitida (ya aplicada)", "file", file)
			continue
		}

		path := filepath.Join(migrationsDir, file)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("leyendo %s: %w", file, err)
		}

		// Cada migración corre en su propia transacción para rollback atómico
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("iniciando transacción para %s: %w", file, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("aplicando %s: %w", file, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", file); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("registrando migración %s: %w", file, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit de %s: %w", file, err)
		}

		slog.Info("migración aplicada", "file", file)
		pendingCount++
	}

	if pendingCount == 0 {
		slog.Info("sin migraciones pendientes — la DB está actualizada")
	} else {
		slog.Info("migraciones aplicadas exitosamente", "count", pendingCount)
	}
	return nil
}

// runDown revierte las últimas N migraciones aplicadas.
// Requiere archivos .down.sql con el mismo prefijo numérico.
func runDown(ctx context.Context, conn *pgx.Conn, count int) error {
	// Obtener las últimas N versiones aplicadas en orden inverso
	rows, err := conn.Query(ctx,
		"SELECT version FROM schema_migrations ORDER BY version DESC LIMIT $1", count)
	if err != nil {
		return fmt.Errorf("obteniendo migraciones para revertir: %w", err)
	}
	defer rows.Close()

	var toRevert []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return err
		}
		toRevert = append(toRevert, version)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(toRevert) == 0 {
		slog.Info("sin migraciones aplicadas para revertir")
		return nil
	}

	for _, version := range toRevert {
		// Buscar archivo .down.sql correspondiente
		// Convención: 001_name.sql → 001_name.down.sql
		downFile := strings.TrimSuffix(version, ".sql") + ".down.sql"
		downPath := filepath.Join(migrationsDir, downFile)

		sql, err := os.ReadFile(downPath)
		if err != nil {
			return fmt.Errorf("archivo de reversión no encontrado para %s (busca %s): %w",
				version, downFile, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("iniciando transacción para down %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("revirtiendo %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx,
			"DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("eliminando registro de %s: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit down de %s: %w", version, err)
		}

		slog.Info("migración revertida", "version", version)
	}
	return nil
}

// showStatus muestra el estado de todas las migraciones.
func showStatus(ctx context.Context, conn *pgx.Conn) error {
	files, err := getMigrationFiles()
	if err != nil {
		return err
	}

	applied, err := getApplied(ctx, conn)
	if err != nil {
		return fmt.Errorf("obteniendo migraciones aplicadas: %w", err)
	}

	fmt.Println("\nEstado de migraciones CitaSpot:")
	fmt.Println(strings.Repeat("─", 55))
	for _, file := range files {
		if applied[file] {
			fmt.Printf("  [✓] %s\n", file)
		} else {
			fmt.Printf("  [ ] %s  ← pendiente\n", file)
		}
	}
	fmt.Println(strings.Repeat("─", 55))
	return nil
}
