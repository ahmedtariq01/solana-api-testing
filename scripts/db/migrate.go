package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"solana-balance-api/internal/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          func(*mongo.Database) error
	Down        func(*mongo.Database) error
}

// MigrationManager handles database migrations
type MigrationManager struct {
	client     *mongo.Client
	db         *mongo.Database
	config     *config.MongoDBConfig
	migrations []Migration
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(cfg *config.MongoDBConfig) (*MigrationManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.URI)
	clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	clientOptions.SetConnectTimeout(cfg.ConnectTimeout)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)

	mm := &MigrationManager{
		client: client,
		db:     db,
		config: cfg,
	}

	// Initialize migrations
	mm.initializeMigrations()

	return mm, nil
}

// initializeMigrations sets up all available migrations
func (mm *MigrationManager) initializeMigrations() {
	mm.migrations = []Migration{
		{
			Version:     1,
			Description: "Create API keys collection with indexes",
			Up:          mm.migration001Up,
			Down:        mm.migration001Down,
		},
		{
			Version:     2,
			Description: "Add last_used field to existing API keys",
			Up:          mm.migration002Up,
			Down:        mm.migration002Down,
		},
		{
			Version:     3,
			Description: "Add compound indexes for performance optimization",
			Up:          mm.migration003Up,
			Down:        mm.migration003Down,
		},
	}
}

// migration001Up creates the API keys collection with basic indexes
func (mm *MigrationManager) migration001Up(db *mongo.Database) error {
	collection := db.Collection(mm.config.APIKeyCollection)

	// Create unique index on key field
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create unique index on key: %w", err)
	}

	log.Println("Migration 001: Created API keys collection with unique key index")
	return nil
}

// migration001Down removes the API keys collection
func (mm *MigrationManager) migration001Down(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := db.Collection(mm.config.APIKeyCollection).Drop(ctx)
	if err != nil {
		return fmt.Errorf("failed to drop API keys collection: %w", err)
	}

	log.Println("Migration 001 rollback: Dropped API keys collection")
	return nil
}

// migration002Up adds last_used field to existing API keys
func (mm *MigrationManager) migration002Up(db *mongo.Database) error {
	collection := db.Collection(mm.config.APIKeyCollection)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Add last_used field to documents that don't have it
	filter := bson.M{"last_used": bson.M{"$exists": false}}
	update := bson.M{"$set": bson.M{"last_used": nil}}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add last_used field: %w", err)
	}

	log.Printf("Migration 002: Added last_used field to %d API keys", result.ModifiedCount)
	return nil
}

// migration002Down removes last_used field from API keys
func (mm *MigrationManager) migration002Down(db *mongo.Database) error {
	collection := db.Collection(mm.config.APIKeyCollection)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Remove last_used field
	filter := bson.M{}
	update := bson.M{"$unset": bson.M{"last_used": ""}}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to remove last_used field: %w", err)
	}

	log.Printf("Migration 002 rollback: Removed last_used field from %d API keys", result.ModifiedCount)
	return nil
}

// migration003Up adds performance optimization indexes
func (mm *MigrationManager) migration003Up(db *mongo.Database) error {
	collection := db.Collection(mm.config.APIKeyCollection)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create index on active field
	activeIndexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "active", Value: 1}},
	}

	_, err := collection.Indexes().CreateOne(ctx, activeIndexModel)
	if err != nil {
		return fmt.Errorf("failed to create active index: %w", err)
	}

	// Create compound index on key and active
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

	log.Println("Migration 003: Added performance optimization indexes")
	return nil
}

// migration003Down removes performance optimization indexes
func (mm *MigrationManager) migration003Down(db *mongo.Database) error {
	collection := db.Collection(mm.config.APIKeyCollection)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Drop active index
	_, err := collection.Indexes().DropOne(ctx, "active_1")
	if err != nil {
		log.Printf("Warning: failed to drop active index: %v", err)
	}

	// Drop compound index
	_, err = collection.Indexes().DropOne(ctx, "key_1_active_1")
	if err != nil {
		log.Printf("Warning: failed to drop compound index: %v", err)
	}

	log.Println("Migration 003 rollback: Removed performance optimization indexes")
	return nil
}

// GetCurrentVersion returns the current migration version
func (mm *MigrationManager) GetCurrentVersion() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := mm.db.Collection("migrations")

	var result struct {
		Version int `bson:"version"`
	}

	err := collection.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil // No migrations have been run
		}
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return result.Version, nil
}

// setVersion records the current migration version
func (mm *MigrationManager) setVersion(version int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := mm.db.Collection("migrations")

	doc := bson.M{
		"version":    version,
		"applied_at": time.Now(),
	}

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}

	return nil
}

// MigrateUp runs all pending migrations
func (mm *MigrationManager) MigrateUp() error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	log.Printf("Current migration version: %d", currentVersion)

	for _, migration := range mm.migrations {
		if migration.Version > currentVersion {
			log.Printf("Running migration %d: %s", migration.Version, migration.Description)

			if err := migration.Up(mm.db); err != nil {
				return fmt.Errorf("migration %d failed: %w", migration.Version, err)
			}

			if err := mm.setVersion(migration.Version); err != nil {
				return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
			}

			log.Printf("Migration %d completed successfully", migration.Version)
		}
	}

	log.Println("All migrations completed successfully")
	return nil
}

// MigrateDown rolls back the last migration
func (mm *MigrationManager) MigrateDown() error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == 0 {
		log.Println("No migrations to roll back")
		return nil
	}

	// Find the migration to roll back
	var targetMigration *Migration
	for _, migration := range mm.migrations {
		if migration.Version == currentVersion {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration %d not found", currentVersion)
	}

	log.Printf("Rolling back migration %d: %s", targetMigration.Version, targetMigration.Description)

	if err := targetMigration.Down(mm.db); err != nil {
		return fmt.Errorf("rollback of migration %d failed: %w", targetMigration.Version, err)
	}

	// Remove the migration record
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := mm.db.Collection("migrations")
	_, err = collection.DeleteOne(ctx, bson.M{"version": currentVersion})
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	log.Printf("Migration %d rolled back successfully", targetMigration.Version)
	return nil
}

// Close closes the database connection
func (mm *MigrationManager) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mm.client.Disconnect(ctx)
}

func RunMigrations() {
	// Load configuration
	cfg := config.LoadConfig()

	// Create migration manager
	manager, err := NewMigrationManager(&cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}
	defer manager.Close()

	// Run migrations
	if err := manager.MigrateUp(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Database migration completed successfully!")
}
func main() {
	RunMigrations()
}
