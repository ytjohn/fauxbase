package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// GetLogger retrieves the logger from the context
func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// Rename to export
type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

// Add this type at the top with other types
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.Status = code
	rw.ResponseWriter.WriteHeader(code)
}

// SetStatus sets the status code without writing the header
func (rw *ResponseWriter) SetStatus(code int) {
	rw.Status = code
	// Remove the WriteHeader call here since Huma will call it
}

// WithLogger adds logging middleware to track requests and errors
func WithLogger(baseLogger *slog.Logger) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		start := time.Now()

		// Set log level based on path
		logLevel := slog.LevelInfo
		if strings.HasPrefix(ctx.Operation().Path, "/favicon") {
			logLevel = slog.LevelDebug
		}

		// Create request-specific logger with base logger
		logger := baseLogger.With(
			"path", ctx.Operation().Path,
			"method", ctx.Operation().Method,
		)

		// Store logger in context
		ctx = huma.WithValue(ctx, LoggerKey, logger)

		// Log incoming request at appropriate level
		logger.Log(ctx.Context(), logLevel, "incoming request")

		// Call next handler
		next(ctx)

		// Log completion with duration
		duration := time.Since(start)
		logger.Log(ctx.Context(), logLevel, "request completed",
			"duration_ms", duration.Milliseconds(),
			"status", ctx.Status(),
		)
	}
}
