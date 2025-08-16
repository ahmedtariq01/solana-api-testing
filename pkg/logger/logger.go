package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ContextKey represents keys used in context for logging
type ContextKey string

const (
	// CorrelationIDKey is the key for correlation ID in context
	CorrelationIDKey ContextKey = "correlation_id"
	// RequestIDKey is the key for request ID in context
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the key for user/API key ID in context
	UserIDKey ContextKey = "user_id"
)

// Logger wraps zap logger with additional functionality
type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// Config represents logger configuration
type Config struct {
	Level       string   `json:"level" default:"info"`
	Environment string   `json:"environment" default:"development"`
	OutputPaths []string `json:"output_paths"`
}

var (
	// Global logger instance
	globalLogger *Logger
)

// Initialize sets up the global logger
func Initialize(config *Config) error {
	var zapConfig zap.Config

	// Configure based on environment
	if config.Environment == "production" {
		zapConfig = zap.NewProductionConfig()
		zapConfig.DisableStacktrace = true
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.DisableStacktrace = false
	}

	// Set log level
	level, err := zap.ParseAtomicLevel(config.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	zapConfig.Level = level

	// Set output paths
	if len(config.OutputPaths) > 0 {
		zapConfig.OutputPaths = config.OutputPaths
	}

	// Add custom fields
	zapConfig.InitialFields = map[string]interface{}{
		"service": "solana-balance-api",
		"version": "1.0.0",
	}

	// Build logger
	zapLogger, err := zapConfig.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %w", err)
	}

	globalLogger = &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}

	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Fallback to development logger if not initialized
		config := &Config{
			Level:       "info",
			Environment: "development",
		}
		if err := Initialize(config); err != nil {
			panic(fmt.Sprintf("failed to initialize fallback logger: %v", err))
		}
	}
	return globalLogger
}

// WithContext creates a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := []zap.Field{}

	// Add correlation ID if present
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		fields = append(fields, zap.String("correlation_id", correlationID.(string)))
	}

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields = append(fields, zap.String("request_id", requestID.(string)))
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		fields = append(fields, zap.String("user_id", userID.(string)))
	}

	return &Logger{
		Logger: l.Logger.With(fields...),
		sugar:  l.Logger.With(fields...).Sugar(),
	}
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return &Logger{
		Logger: l.Logger.With(zapFields...),
		sugar:  l.Logger.With(zapFields...).Sugar(),
	}
}

// WithError creates a logger with error field
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Error(err)),
		sugar:  l.Logger.With(zap.Error(err)).Sugar(),
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// Infof logs an info message with formatting
func (l *Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

// Warnf logs a warning message with formatting
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

// Errorf logs an error message with formatting
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

// Debugf logs a debug message with formatting
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

// Fatalf logs a fatal message with formatting and exits
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Close closes the logger
func (l *Logger) Close() error {
	return l.Logger.Sync()
}

// GenerateCorrelationID generates a new correlation ID
func GenerateCorrelationID() string {
	return uuid.New().String()
}

// GenerateRequestID generates a new request ID
func GenerateRequestID() string {
	return uuid.New().String()
}

// ContextWithCorrelationID adds correlation ID to context
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// ContextWithRequestID adds request ID to context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// ContextWithUserID adds user ID to context
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetCorrelationIDFromContext extracts correlation ID from context
func GetCorrelationIDFromContext(ctx context.Context) string {
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		return correlationID.(string)
	}
	return ""
}

// GetRequestIDFromContext extracts request ID from context
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		return requestID.(string)
	}
	return ""
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) string {
	if userID := ctx.Value(UserIDKey); userID != nil {
		return userID.(string)
	}
	return ""
}

// LoggingMiddleware creates a Gin middleware for structured logging with correlation IDs
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate correlation and request IDs
		correlationID := GenerateCorrelationID()
		requestID := GenerateRequestID()

		// Add IDs to Gin context
		c.Set(string(CorrelationIDKey), correlationID)
		c.Set(string(RequestIDKey), requestID)

		// Add IDs to request context
		ctx := c.Request.Context()
		ctx = ContextWithCorrelationID(ctx, correlationID)
		ctx = ContextWithRequestID(ctx, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Add correlation ID to response headers
		c.Header("X-Correlation-ID", correlationID)
		c.Header("X-Request-ID", requestID)

		// Create logger with context
		logger := GetLogger().WithContext(ctx)

		// Log request start
		logger.Info("Request started",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("client_ip", c.ClientIP()),
			zap.String("remote_addr", c.Request.RemoteAddr),
		)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Determine log level based on status code
		statusCode := c.Writer.Status()
		logLevel := zap.InfoLevel
		if statusCode >= 400 && statusCode < 500 {
			logLevel = zap.WarnLevel
		} else if statusCode >= 500 {
			logLevel = zap.ErrorLevel
		}

		// Log request completion
		switch logLevel {
		case zap.ErrorLevel:
			logger.Error("Request completed",
				zap.Int("status_code", statusCode),
				zap.Duration("duration", duration),
				zap.Int("response_size", c.Writer.Size()),
			)
		case zap.WarnLevel:
			logger.Warn("Request completed",
				zap.Int("status_code", statusCode),
				zap.Duration("duration", duration),
				zap.Int("response_size", c.Writer.Size()),
			)
		default:
			logger.Info("Request completed",
				zap.Int("status_code", statusCode),
				zap.Duration("duration", duration),
				zap.Int("response_size", c.Writer.Size()),
			)
		}

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.Error("Request error",
					zap.Uint64("error_type", uint64(err.Type)),
					zap.Error(err.Err),
				)
			}
		}
	}
}

// RecoveryMiddleware creates a Gin middleware for panic recovery with logging
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Get logger with context
		ctx := c.Request.Context()
		logger := GetLogger().WithContext(ctx)

		// Log the panic
		logger.Error("Panic recovered",
			zap.Any("panic", recovered),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
		)

		// Return 500 error
		c.JSON(500, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Internal server error",
				"details": "An unexpected error occurred",
			},
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
			"correlation_id": GetCorrelationIDFromContext(ctx),
		})
	})
}
