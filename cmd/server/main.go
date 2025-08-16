package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/handlers"
	"solana-balance-api/internal/middleware"
	"solana-balance-api/internal/services"
	"solana-balance-api/pkg/logger"
	"solana-balance-api/pkg/ratelimiter"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the main application server
type Server struct {
	httpServer     *http.Server
	config         *config.Config
	authService    *services.AuthService
	solanaClient   *services.SolanaClient
	balanceService *services.BalanceService
	rateLimiter    *ratelimiter.RateLimiter
	router         *handlers.Router
}

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logging
	loggerConfig := &logger.Config{
		Level:       cfg.Logging.Level,
		Environment: cfg.Logging.Environment,
		OutputPaths: cfg.Logging.OutputPaths,
	}

	if err := logger.Initialize(loggerConfig); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log := logger.GetLogger()

	log.Info("Starting Solana Balance API server",
		zap.String("host", cfg.Server.Host),
		zap.String("port", cfg.Server.Port),
		zap.String("mongodb_uri", cfg.MongoDB.URI),
		zap.String("rpc_endpoint", cfg.RPC.Endpoint),
		zap.Duration("cache_ttl", cfg.Cache.TTL),
		zap.Int("rate_limit_rpm", cfg.RateLimit.RequestsPerMinute),
		zap.String("log_level", cfg.Logging.Level),
		zap.String("environment", cfg.Logging.Environment),
	)

	// Initialize and start server
	server, err := NewServer(cfg)
	if err != nil {
		log.Fatal("Failed to create server", zap.Error(err))
	}

	// Start server with graceful shutdown
	if err := server.Start(); err != nil {
		log.Fatal("Server failed to start", zap.Error(err))
	}
}

// NewServer creates a new server instance with all dependencies
func NewServer(cfg *config.Config) (*Server, error) {
	log := logger.GetLogger()

	log.Info("Initializing server components")

	// Initialize authentication service
	log.Debug("Initializing authentication service")
	authService, err := services.NewAuthService(&cfg.MongoDB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth service: %w", err)
	}

	// Initialize Solana RPC client
	log.Debug("Initializing Solana RPC client")
	solanaClient := services.NewSolanaClient(&cfg.RPC)

	// Test RPC connection
	log.Debug("Testing RPC connection health")
	if err := solanaClient.IsHealthy(); err != nil {
		log.Warn("Solana RPC health check failed", zap.Error(err))
	} else {
		log.Info("Solana RPC connection healthy")
	}

	// Initialize balance service
	log.Debug("Initializing balance service")
	balanceService := services.NewBalanceService(solanaClient, cfg)

	// Initialize rate limiter
	log.Debug("Initializing rate limiter")
	rateLimiter := ratelimiter.New(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.WindowSize)

	// Initialize database health checker
	log.Debug("Initializing database health checker")
	dbHealthChecker, err := services.NewDatabaseHealthChecker(&cfg.MongoDB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database health checker: %w", err)
	}

	// Initialize health handler
	log.Debug("Initializing health handler")
	healthHandler := handlers.NewHealthHandler(dbHealthChecker)

	// Initialize router
	log.Debug("Initializing router")
	router := handlers.NewRouter(balanceService, healthHandler)

	log.Info("Server components initialized successfully")

	return &Server{
		config:         cfg,
		authService:    authService,
		solanaClient:   solanaClient,
		balanceService: balanceService,
		rateLimiter:    rateLimiter,
		router:         router,
	}, nil
}

// Start starts the HTTP server with graceful shutdown handling
func (s *Server) Start() error {
	log := logger.GetLogger()

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Debug("Creating Gin engine")

	// Create Gin engine
	engine := gin.New()

	// Setup middleware stack
	s.setupMiddleware(engine)

	// Setup routes
	s.setupRoutes(engine)

	// Create HTTP server with optimized timeout configurations
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port),
		Handler:      engine,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
		// Additional performance optimizations
		ReadHeaderTimeout: 5 * time.Second, // Prevent slow header attacks
		MaxHeaderBytes:    1 << 20,         // 1MB max header size
		// Enable HTTP/2 for better performance
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Info("HTTP server configured",
		zap.String("address", s.httpServer.Addr),
		zap.Duration("read_timeout", s.config.Server.ReadTimeout),
		zap.Duration("write_timeout", s.config.Server.WriteTimeout),
		zap.Duration("idle_timeout", s.config.Server.IdleTimeout),
	)

	// Start cleanup routines
	s.startCleanupRoutines()

	// Start server in a goroutine
	go func() {
		log.Info("Starting HTTP server", zap.String("address", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	return s.waitForShutdown()
}

// setupMiddleware configures the middleware stack
func (s *Server) setupMiddleware(engine *gin.Engine) {
	log := logger.GetLogger()

	log.Debug("Setting up middleware stack")

	// Recovery middleware with structured logging (should be first)
	engine.Use(logger.RecoveryMiddleware())

	// Structured logging middleware with correlation IDs
	engine.Use(logger.LoggingMiddleware())

	// Performance monitoring middleware stack
	engine.Use(middleware.PerformanceMiddleware(s.balanceService.GetMetricsCollector()))
	engine.Use(middleware.RequestSizeMiddleware())
	engine.Use(middleware.ConcurrencyMiddleware(s.balanceService.GetMetricsCollector()))

	// Metrics middleware to track performance
	engine.Use(middleware.MetricsMiddleware(s.balanceService.GetMetricsCollector()))

	// CORS middleware (if needed)
	engine.Use(s.corsMiddleware())

	// Rate limiting middleware (before auth to prevent auth bypass attempts)
	engine.Use(s.rateLimiter.Middleware())

	log.Debug("Middleware stack configured")
}

// setupRoutes configures all application routes
func (s *Server) setupRoutes(engine *gin.Engine) {
	// Health check routes (no authentication required)
	s.router.SetupHealthRoutes(engine)

	// API routes with authentication
	api := engine.Group("/api")
	api.Use(middleware.AuthMiddleware(s.authService))
	{
		// Balance endpoints
		api.POST("/get-balance", s.router.GetBalanceHandler().GetBalance)
	}

	// Additional monitoring endpoints
	engine.GET("/metrics", s.metricsHandler)
	engine.GET("/status", s.statusHandler)
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// metricsHandler provides comprehensive metrics endpoint
func (s *Server) metricsHandler(c *gin.Context) {
	performanceStats := s.balanceService.GetPerformanceStats()
	c.JSON(http.StatusOK, gin.H{
		"service":     "solana-balance-api",
		"version":     "1.0.0",
		"performance": performanceStats,
	})
}

// statusHandler provides detailed status information
func (s *Server) statusHandler(c *gin.Context) {
	// Check RPC health
	rpcHealthy := true
	if err := s.solanaClient.IsHealthy(); err != nil {
		rpcHealthy = false
	}

	c.JSON(http.StatusOK, gin.H{
		"service":     "solana-balance-api",
		"status":      "running",
		"rpc_healthy": rpcHealthy,
		"uptime":      time.Since(startTime).String(),
		"version":     "1.0.0",
	})
}

// startCleanupRoutines starts background cleanup tasks
func (s *Server) startCleanupRoutines() {
	log := logger.GetLogger()

	// Rate limiter cleanup
	go func() {
		ticker := time.NewTicker(s.config.RateLimit.CleanupInterval)
		defer ticker.Stop()

		log.Debug("Starting rate limiter cleanup routine",
			zap.Duration("interval", s.config.RateLimit.CleanupInterval),
		)

		for range ticker.C {
			s.rateLimiter.Cleanup()
		}
	}()

	log.Info("Background cleanup routines started")
}

// waitForShutdown waits for interrupt signal and performs graceful shutdown
func (s *Server) waitForShutdown() error {
	log := logger.GetLogger()

	// Create channel to receive OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal received
	sig := <-quit
	log.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Info("Shutting down HTTP server", zap.Duration("timeout", 30*time.Second))

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	// Cleanup services
	s.cleanup()

	log.Info("Server gracefully stopped")
	return nil
}

// cleanup performs cleanup of all services
func (s *Server) cleanup() {
	log := logger.GetLogger()

	log.Info("Cleaning up services...")

	// Stop balance service
	if s.balanceService != nil {
		log.Debug("Stopping balance service")
		s.balanceService.Stop()
	}

	// Close auth service (MongoDB connection)
	if s.authService != nil {
		log.Debug("Closing auth service")
		if err := s.authService.Close(); err != nil {
			log.Error("Error closing auth service", zap.Error(err))
		}
	}

	// Sync logger before exit
	if err := logger.GetLogger().Sync(); err != nil {
		// Don't log this error as logger might be closed
		fmt.Printf("Error syncing logger: %v\n", err)
	}

	log.Info("Cleanup completed")
}

// Global variable to track server start time for uptime calculation
var startTime = time.Now()
