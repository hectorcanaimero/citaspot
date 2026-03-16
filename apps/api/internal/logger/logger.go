package logger

import (
	"log/slog"
	"os"
)

// Init configura el logger global (slog.Default).
// En producción usa JSON; en desarrollo usa texto legible.
func Init(env string) {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

// Fatal loggea a nivel Error y termina el proceso.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
