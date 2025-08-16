package examples

import (
	"log"
	"net/http"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/middleware"
	"solana-balance-api/internal/services"

	"github.com/gin-gonic/gin"
)

func AuthUsageExample() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize authentication service
	authService, err := services.NewAuthService(&cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}
	defer authService.Close()

	// Setup Gin router
	router := gin.Default()

	// Apply authentication middleware to protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(authService))

	// Example protected endpoint
	protected.POST("/get-balance", func(c *gin.Context) {
		// Get API key info from context (set by middleware)
		apiKeyName, _ := c.Get("api_key_name")

		c.JSON(http.StatusOK, gin.H{
			"message":          "Balance endpoint accessed successfully",
			"authenticated_as": apiKeyName,
		})
	})

	// Public health check endpoint (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Start server
	log.Printf("Starting server on %s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Fatal(http.ListenAndServe(cfg.Server.Host+":"+cfg.Server.Port, router))
}
