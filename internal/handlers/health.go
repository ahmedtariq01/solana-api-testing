package handlers

import (
	"net/http"
	"time"

	"solana-balance-api/internal/services"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	dbHealthChecker *services.DatabaseHealthChecker
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(dbHealthChecker *services.DatabaseHealthChecker) *HealthHandler {
	return &HealthHandler{
		dbHealthChecker: dbHealthChecker,
	}
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    services.HealthStatus            `json:"status"`
	Timestamp time.Time                        `json:"timestamp"`
	Services  map[string]*services.HealthCheck `json:"services"`
	Version   string                           `json:"version,omitempty"`
}

// GetHealth returns the overall health status
func (h *HealthHandler) GetHealth(c *gin.Context) {
	// Get detailed health information
	serviceChecks := h.dbHealthChecker.GetDetailedHealth()

	// Determine overall status
	overallStatus := services.HealthStatusHealthy
	for _, check := range serviceChecks {
		if check.Status == services.HealthStatusUnhealthy {
			overallStatus = services.HealthStatusUnhealthy
			break
		} else if check.Status == services.HealthStatusDegraded && overallStatus == services.HealthStatusHealthy {
			overallStatus = services.HealthStatusDegraded
		}
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Services:  serviceChecks,
		Version:   "1.0.0", // This could be injected from build info
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if overallStatus == services.HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == services.HealthStatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	c.JSON(statusCode, response)
}

// GetLiveness returns a simple liveness check
func (h *HealthHandler) GetLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

// GetReadiness returns readiness status (checks if all dependencies are available)
func (h *HealthHandler) GetReadiness(c *gin.Context) {
	// Check database connectivity
	dbHealth := h.dbHealthChecker.CheckHealth()

	if dbHealth.Status == services.HealthStatusUnhealthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not_ready",
			"message":   "database not available",
			"timestamp": time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now(),
	})
}

// GetDatabaseHealth returns detailed database health information
func (h *HealthHandler) GetDatabaseHealth(c *gin.Context) {
	healthCheck := h.dbHealthChecker.CheckHealth()

	statusCode := http.StatusOK
	if healthCheck.Status == services.HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, healthCheck)
}
