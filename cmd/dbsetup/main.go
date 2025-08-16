package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/models"
	"solana-balance-api/internal/services"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	var (
		initDB      = flag.Bool("init", false, "Initialize database with schema and indexes")
		seedData    = flag.Bool("seed", false, "Seed database with test data")
		migrate     = flag.Bool("migrate", false, "Run database migrations")
		rollback    = flag.Bool("rollback", false, "Rollback last migration")
		healthCheck = flag.Bool("health", false, "Run database health check")
		all         = flag.Bool("all", false, "Run init, migrate, and seed (full setup)")
	)
	flag.Parse()

	// Load configuration
	cfg := config.LoadConfig()

	// If no flags specified, show usage
	if !*initDB && !*seedData && !*migrate && !*rollback && !*healthCheck && !*all {
		fmt.Println("Database Setup Utility")
		fmt.Println("Usage:")
		fmt.Println("  -init      Initialize database with schema and indexes")
		fmt.Println("  -seed      Seed database with test data")
		fmt.Println("  -migrate   Run database migrations")
		fmt.Println("  -rollback  Rollback last migration")
		fmt.Println("  -health    Run database health check")
		fmt.Println("  -all       Run full setup (init + migrate + seed)")
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  MONGODB_URI              MongoDB connection string")
		fmt.Println("  MONGODB_DATABASE         Database name")
		fmt.Println("  MONGODB_APIKEY_COLLECTION API keys collection name")
		os.Exit(1)
	}

	// Run health check
	if *healthCheck || *all {
		if err := runHealthCheck(&cfg.MongoDB); err != nil {
			log.Fatalf("Health check failed: %v", err)
		}
	}

	// Run migrations
	if *migrate || *all {
		if err := runMigrations(&cfg.MongoDB); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	// Rollback migration
	if *rollback {
		if err := rollbackMigration(&cfg.MongoDB); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
	}

	// Initialize database
	if *initDB || *all {
		if err := initializeDatabase(&cfg.MongoDB); err != nil {
			log.Fatalf("Database initialization failed: %v", err)
		}
	}

	// Seed test data
	if *seedData || *all {
		if err := seedTestData(&cfg.MongoDB); err != nil {
			log.Fatalf("Data seeding failed: %v", err)
		}
	}

	log.Println("Database setup completed successfully!")
}

// runHealthCheck performs a comprehensive health check
func runHealthCheck(cfg *config.MongoDBConfig) error {
	log.Println("Running database health check...")

	healthChecker, err := services.NewDatabaseHealthChecker(cfg)
	if err != nil {
		return fmt.Errorf("failed to create health checker: %w", err)
	}
	defer healthChecker.Close()

	// Get detailed health information
	healthChecks := healthChecker.GetDetailedHealth()

	log.Println("Health Check Results:")
	for service, check := range healthChecks {
		status := "✓"
		if check.Status != services.HealthStatusHealthy {
			status = "✗"
		}
		log.Printf("  %s %s: %s (%v)", status, service, check.Status, check.ResponseTime)
		if check.Message != "" {
			log.Printf("    %s", check.Message)
		}
	}

	// Check if any service is unhealthy
	for _, check := range healthChecks {
		if check.Status == services.HealthStatusUnhealthy {
			return fmt.Errorf("health check failed for %s", check.Service)
		}
	}

	log.Println("All health checks passed!")
	return nil
}

// runMigrations executes database migrations
func runMigrations(cfg *config.MongoDBConfig) error {
	log.Println("Running database migrations...")

	// Note: This would use the migration manager from scripts/db/migrate.go
	// For now, we'll create a simple version here
	initializer, err := NewDatabaseInitializer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database initializer: %w", err)
	}
	defer initializer.Close()

	if err := initializer.InitializeDatabase(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed successfully!")
	return nil
}

// rollbackMigration rolls back the last migration
func rollbackMigration(cfg *config.MongoDBConfig) error {
	log.Println("Rolling back last migration...")

	// This would use the migration manager to rollback
	// For now, we'll just log that it's not implemented
	log.Println("Migration rollback not implemented in this version")
	return nil
}

// initializeDatabase sets up the database schema
func initializeDatabase(cfg *config.MongoDBConfig) error {
	log.Println("Initializing database...")

	initializer, err := NewDatabaseInitializer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database initializer: %w", err)
	}
	defer initializer.Close()

	if err := initializer.InitializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	log.Println("Database initialization completed successfully!")
	return nil
}

// seedTestData creates sample data for testing
func seedTestData(cfg *config.MongoDBConfig) error {
	log.Println("Seeding test data...")

	initializer, err := NewDatabaseInitializer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database initializer: %w", err)
	}
	defer initializer.Close()

	if err := initializer.SeedTestData(); err != nil {
		return fmt.Errorf("failed to seed test data: %w", err)
	}

	log.Println("Test data seeding completed successfully!")
	return nil
}

// DatabaseInitializer handles database setup and seeding
type DatabaseInitializer struct {
	client *mongo.Client
	db     *mongo.Database
	config *config.MongoDBConfig
}

// NewDatabaseInitializer creates a new database initializer
func NewDatabaseInitializer(cfg *config.MongoDBConfig) (*DatabaseInitializer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.URI)
	clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	clientOptions.SetConnectTimeout(cfg.ConnectTimeout)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)

	return &DatabaseInitializer{
		client: client,
		db:     db,
		config: cfg,
	}, nil
}

// InitializeDatabase sets up the database with required collections and indexes
func (di *DatabaseInitializer) InitializeDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Setting up database schema...")

	// Create API keys collection if it doesn't exist
	collection := di.db.Collection(di.config.APIKeyCollection)

	// Create unique index on key field
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create index on key field: %w", err)
	}

	// Create index on active field for faster queries
	activeIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "active", Value: 1}},
	}

	_, err = collection.Indexes().CreateOne(ctx, activeIndexModel)
	if err != nil {
		return fmt.Errorf("failed to create index on active field: %w", err)
	}

	// Create compound index on key and active for optimal query performance
	compoundIndexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "key", Value: 1},
			{Key: "active", Value: 1},
		},
	}

	_, err = collection.Indexes().CreateOne(ctx, compoundIndexModel)
	if err != nil {
		return fmt.Errorf("failed to create compound index: %w", err)
	}

	log.Println("Database schema setup completed successfully")
	return nil
}

// SeedTestData creates sample API keys for testing
func (di *DatabaseInitializer) SeedTestData() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Creating test API keys...")

	collection := di.db.Collection(di.config.APIKeyCollection)

	// Check if test data already exists
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count existing documents: %w", err)
	}

	if count > 0 {
		log.Printf("Found %d existing API keys, skipping seed data creation", count)
		return nil
	}

	// Create sample API keys
	testAPIKeys := []models.APIKey{
		{
			Key:       "test-api-key-1",
			Name:      "Test API Key 1",
			Active:    true,
			CreatedAt: time.Now(),
		},
		{
			Key:       "test-api-key-2",
			Name:      "Test API Key 2",
			Active:    true,
			CreatedAt: time.Now(),
		},
		{
			Key:       "inactive-test-key",
			Name:      "Inactive Test Key",
			Active:    false,
			CreatedAt: time.Now(),
		},
	}

	// Generate additional random API keys for load testing
	for i := 0; i < 5; i++ {
		randomKey, err := generateRandomAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate random API key: %w", err)
		}

		testAPIKeys = append(testAPIKeys, models.APIKey{
			Key:       randomKey,
			Name:      fmt.Sprintf("Generated Test Key %d", i+1),
			Active:    true,
			CreatedAt: time.Now(),
		})
	}

	// Insert test API keys
	var documents []interface{}
	for _, apiKey := range testAPIKeys {
		documents = append(documents, apiKey)
	}

	result, err := collection.InsertMany(ctx, documents)
	if err != nil {
		return fmt.Errorf("failed to insert test API keys: %w", err)
	}

	log.Printf("Successfully created %d test API keys", len(result.InsertedIDs))

	// Print the test API keys for reference
	log.Println("Test API Keys created:")
	for _, apiKey := range testAPIKeys {
		status := "active"
		if !apiKey.Active {
			status = "inactive"
		}
		log.Printf("  - %s (%s) [%s]", apiKey.Key, apiKey.Name, status)
	}

	return nil
}

// generateRandomAPIKey generates a cryptographically secure random API key
func generateRandomAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Close closes the database connection
func (di *DatabaseInitializer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return di.client.Disconnect(ctx)
}
