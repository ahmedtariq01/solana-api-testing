package middleware

import (
	"strings"

	"solana-balance-api/internal/models"
	"solana-balance-api/internal/services"
	"solana-balance-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware creates a middleware for API key authentication
func AuthMiddleware(authService services.AuthServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get logger with context
		log := logger.GetLogger().WithContext(c.Request.Context())

		log.Debug("Authenticating request",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)

		// Get API key from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("Missing API key in Authorization header",
				zap.String("client_ip", c.ClientIP()),
				zap.String("user_agent", c.Request.UserAgent()),
			)

			appErr := models.NewAppErrorWithDetails(
				models.ErrorCodeMissingAPIKey,
				"API key is required",
				"Provide API key in Authorization header",
			)
			models.HandleError(c, appErr, log)
			c.Abort()
			return
		}

		// Extract API key from "Bearer <key>" or just "<key>"
		apiKey := strings.TrimSpace(authHeader)
		if strings.HasPrefix(strings.ToLower(apiKey), "bearer") {
			// Handle both "Bearer <key>" and "Bearer<key>" formats
			if len(apiKey) > 6 {
				apiKey = strings.TrimSpace(apiKey[6:])
			} else {
				apiKey = ""
			}
		}

		if apiKey == "" {
			log.Warn("Empty API key after parsing Authorization header",
				zap.String("auth_header_format", authHeader),
				zap.String("client_ip", c.ClientIP()),
			)

			appErr := models.NewAppErrorWithDetails(
				models.ErrorCodeInvalidAPIKey,
				"Invalid API key format",
				"API key cannot be empty",
			)
			models.HandleError(c, appErr, log)
			c.Abort()
			return
		}

		// Validate API key (don't log the actual key for security)
		log.Debug("Validating API key with auth service")

		validatedKey, err := authService.ValidateAPIKey(apiKey)
		if err != nil {
			log.Warn("API key validation failed",
				zap.Error(err),
				zap.String("client_ip", c.ClientIP()),
			)

			var appErr *models.AppError
			switch err {
			case services.ErrInvalidAPIKey:
				appErr = models.NewAppError(models.ErrorCodeInvalidAPIKey, "Invalid API key")
			case services.ErrInactiveAPIKey:
				appErr = models.NewAppError(models.ErrorCodeInactiveAPIKey, "API key is inactive")
			case services.ErrDatabaseError:
				appErr = models.NewAppErrorWithCause(models.ErrorCodeDatabaseError, "Authentication service unavailable", err)
			default:
				appErr = models.NewAppErrorWithCause(models.ErrorCodeInvalidAPIKey, "Authentication failed", err)
			}

			models.HandleError(c, appErr, log)
			c.Abort()
			return
		}

		// Store validated API key in context for use in handlers
		c.Set("api_key", validatedKey)
		c.Set("api_key_id", validatedKey.ID.Hex())
		c.Set("api_key_name", validatedKey.Name)

		// Add user ID to request context for logging
		ctx := logger.ContextWithUserID(c.Request.Context(), validatedKey.ID.Hex())
		c.Request = c.Request.WithContext(ctx)

		log.Info("Authentication successful",
			zap.String("api_key_id", validatedKey.ID.Hex()),
			zap.String("api_key_name", validatedKey.Name),
		)

		c.Next()
	}
}
