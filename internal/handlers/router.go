package handlers

import (
	"solana-balance-api/internal/services"

	"github.com/gin-gonic/gin"
)

// Router handles HTTP routing setup
type Router struct {
	balanceHandler *BalanceHandler
	healthHandler  *HealthHandler
}

// NewRouter creates a new Router instance with all handlers
func NewRouter(balanceService services.BalanceServiceInterface, healthHandler *HealthHandler) *Router {
	return &Router{
		balanceHandler: NewBalanceHandler(balanceService),
		healthHandler:  healthHandler,
	}
}

// GetBalanceHandler returns the balance handler for external access
func (r *Router) GetBalanceHandler() *BalanceHandler {
	return r.balanceHandler
}

// SetupRoutes configures all API routes
func (r *Router) SetupRoutes(engine *gin.Engine) {
	// API v1 routes
	api := engine.Group("/api")
	{
		// Balance endpoints
		api.POST("/get-balance", r.balanceHandler.GetBalance)
	}
}

// SetupHealthRoutes configures health check routes
func (r *Router) SetupHealthRoutes(engine *gin.Engine) {
	// Health check endpoints
	health := engine.Group("/health")
	{
		health.GET("", r.healthHandler.GetHealth)            // Overall health
		health.GET("/live", r.healthHandler.GetLiveness)     // Liveness probe
		health.GET("/ready", r.healthHandler.GetReadiness)   // Readiness probe
		health.GET("/db", r.healthHandler.GetDatabaseHealth) // Database health
	}
}
