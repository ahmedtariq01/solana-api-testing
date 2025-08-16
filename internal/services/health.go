package services

import (
	"context"
	"fmt"
	"time"

	"solana-balance-api/internal/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// HealthCheck represents a health check result
type HealthCheck struct {
	Service      string        `json:"service"`
	Status       HealthStatus  `json:"status"`
	Message      string        `json:"message,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	Timestamp    time.Time     `json:"timestamp"`
}

// DatabaseHealthChecker provides health check functionality for MongoDB
type DatabaseHealthChecker struct {
	client *mongo.Client
	db     *mongo.Database
	config *config.MongoDBConfig
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(cfg *config.MongoDBConfig) (*DatabaseHealthChecker, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.URI)
	clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	clientOptions.SetConnectTimeout(cfg.ConnectTimeout)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)

	return &DatabaseHealthChecker{
		client: client,
		db:     db,
		config: cfg,
	}, nil
}

// CheckHealth performs a comprehensive health check of the MongoDB connection
func (dhc *DatabaseHealthChecker) CheckHealth() *HealthCheck {
	start := time.Now()

	healthCheck := &HealthCheck{
		Service:   "mongodb",
		Timestamp: start,
	}

	// Test basic connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dhc.client.Ping(ctx, nil); err != nil {
		healthCheck.Status = HealthStatusUnhealthy
		healthCheck.Message = fmt.Sprintf("ping failed: %v", err)
		healthCheck.ResponseTime = time.Since(start)
		return healthCheck
	}

	// Test database operations
	if err := dhc.testDatabaseOperations(ctx); err != nil {
		healthCheck.Status = HealthStatusDegraded
		healthCheck.Message = fmt.Sprintf("database operations failed: %v", err)
		healthCheck.ResponseTime = time.Since(start)
		return healthCheck
	}

	// Test collection access
	if err := dhc.testCollectionAccess(ctx); err != nil {
		healthCheck.Status = HealthStatusDegraded
		healthCheck.Message = fmt.Sprintf("collection access failed: %v", err)
		healthCheck.ResponseTime = time.Since(start)
		return healthCheck
	}

	healthCheck.Status = HealthStatusHealthy
	healthCheck.Message = "all checks passed"
	healthCheck.ResponseTime = time.Since(start)

	return healthCheck
}

// testDatabaseOperations tests basic database operations
func (dhc *DatabaseHealthChecker) testDatabaseOperations(ctx context.Context) error {
	// Test database stats
	var result bson.M
	err := dhc.db.RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	return nil
}

// testCollectionAccess tests access to the API keys collection
func (dhc *DatabaseHealthChecker) testCollectionAccess(ctx context.Context) error {
	collection := dhc.db.Collection(dhc.config.APIKeyCollection)

	// Test collection stats
	var result bson.M
	err := dhc.db.RunCommand(ctx, bson.D{
		{Key: "collStats", Value: dhc.config.APIKeyCollection},
	}).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to get collection stats: %w", err)
	}

	// Test a simple count operation
	_, err = collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}

	return nil
}

// CheckConnectionPool checks the health of the MongoDB connection pool
func (dhc *DatabaseHealthChecker) CheckConnectionPool() *HealthCheck {
	start := time.Now()

	healthCheck := &HealthCheck{
		Service:   "mongodb_pool",
		Timestamp: start,
	}

	// Get server status to check connection pool stats
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result bson.M
	err := dhc.db.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result)
	if err != nil {
		healthCheck.Status = HealthStatusUnhealthy
		healthCheck.Message = fmt.Sprintf("failed to get server status: %v", err)
		healthCheck.ResponseTime = time.Since(start)
		return healthCheck
	}

	// Check if connections section exists
	if connections, ok := result["connections"].(bson.M); ok {
		current, currentOk := connections["current"].(int32)
		available, availableOk := connections["available"].(int32)

		if currentOk && availableOk {
			if available < 10 { // Less than 10 available connections
				healthCheck.Status = HealthStatusDegraded
				healthCheck.Message = fmt.Sprintf("low available connections: %d current, %d available", current, available)
			} else {
				healthCheck.Status = HealthStatusHealthy
				healthCheck.Message = fmt.Sprintf("connection pool healthy: %d current, %d available", current, available)
			}
		} else {
			healthCheck.Status = HealthStatusDegraded
			healthCheck.Message = "unable to parse connection stats"
		}
	} else {
		healthCheck.Status = HealthStatusDegraded
		healthCheck.Message = "connection stats not available"
	}

	healthCheck.ResponseTime = time.Since(start)
	return healthCheck
}

// CheckIndexes verifies that required indexes exist
func (dhc *DatabaseHealthChecker) CheckIndexes() *HealthCheck {
	start := time.Now()

	healthCheck := &HealthCheck{
		Service:   "mongodb_indexes",
		Timestamp: start,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := dhc.db.Collection(dhc.config.APIKeyCollection)

	// Get all indexes
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		healthCheck.Status = HealthStatusUnhealthy
		healthCheck.Message = fmt.Sprintf("failed to list indexes: %v", err)
		healthCheck.ResponseTime = time.Since(start)
		return healthCheck
	}
	defer cursor.Close(ctx)

	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		healthCheck.Status = HealthStatusUnhealthy
		healthCheck.Message = fmt.Sprintf("failed to decode indexes: %v", err)
		healthCheck.ResponseTime = time.Since(start)
		return healthCheck
	}

	// Check for required indexes
	requiredIndexes := map[string]bool{
		"key_1":          false, // Unique index on key field
		"active_1":       false, // Index on active field
		"key_1_active_1": false, // Compound index
	}

	for _, index := range indexes {
		if name, ok := index["name"].(string); ok {
			if _, required := requiredIndexes[name]; required {
				requiredIndexes[name] = true
			}
		}
	}

	// Check if all required indexes exist
	var missingIndexes []string
	for indexName, exists := range requiredIndexes {
		if !exists {
			missingIndexes = append(missingIndexes, indexName)
		}
	}

	if len(missingIndexes) > 0 {
		healthCheck.Status = HealthStatusDegraded
		healthCheck.Message = fmt.Sprintf("missing indexes: %v", missingIndexes)
	} else {
		healthCheck.Status = HealthStatusHealthy
		healthCheck.Message = "all required indexes present"
	}

	healthCheck.ResponseTime = time.Since(start)
	return healthCheck
}

// GetDetailedHealth returns comprehensive health information
func (dhc *DatabaseHealthChecker) GetDetailedHealth() map[string]*HealthCheck {
	return map[string]*HealthCheck{
		"connectivity":    dhc.CheckHealth(),
		"connection_pool": dhc.CheckConnectionPool(),
		"indexes":         dhc.CheckIndexes(),
	}
}

// Close closes the database connection
func (dhc *DatabaseHealthChecker) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return dhc.client.Disconnect(ctx)
}
