package services

import (
	"context"
	"errors"
	"time"

	"solana-balance-api/internal/config"
	"solana-balance-api/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	ErrInvalidAPIKey  = errors.New("invalid API key")
	ErrInactiveAPIKey = errors.New("API key is inactive")
	ErrDatabaseError  = errors.New("database error")
)

// AuthService handles API key authentication using MongoDB
type AuthService struct {
	db         *mongo.Database
	collection *mongo.Collection
	config     *config.MongoDBConfig
}

// NewAuthService creates a new authentication service with optimized MongoDB connection
func NewAuthService(cfg *config.MongoDBConfig) (*AuthService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	// Set optimized client options with enhanced connection pooling
	clientOptions := options.Client().ApplyURI(cfg.URI)

	// Connection pool optimization
	clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	clientOptions.SetMinPoolSize(uint64(cfg.MaxPoolSize / 4)) // 25% of max as minimum
	clientOptions.SetMaxConnIdleTime(30 * time.Minute)
	clientOptions.SetMaxConnecting(uint64(cfg.MaxPoolSize / 2)) // Limit concurrent connections

	// Timeout configurations for performance
	clientOptions.SetConnectTimeout(cfg.ConnectTimeout)
	clientOptions.SetSocketTimeout(30 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)
	clientOptions.SetHeartbeatInterval(10 * time.Second)

	// Network optimization
	clientOptions.SetCompressors([]string{"snappy", "zlib", "zstd"})

	// Read preference for better load distribution
	clientOptions.SetReadPreference(readpref.SecondaryPreferred())

	// Enable retryable writes for better reliability
	clientOptions.SetRetryWrites(true)
	clientOptions.SetRetryReads(true)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(cfg.Database)
	collection := db.Collection(cfg.APIKeyCollection)

	// Create index on key field for fast lookups
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		// Index might already exist, which is fine
		// We'll continue without failing
	}

	return &AuthService{
		db:         db,
		collection: collection,
		config:     cfg,
	}, nil
}

// ValidateAPIKey validates an API key against the MongoDB database
func (a *AuthService) ValidateAPIKey(key string) (*models.APIKey, error) {
	if key == "" {
		return nil, ErrInvalidAPIKey
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var apiKey models.APIKey
	filter := bson.M{"key": key}

	err := a.collection.FindOne(ctx, filter).Decode(&apiKey)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrInvalidAPIKey
		}
		return nil, ErrDatabaseError
	}

	// Check if API key is active
	if !apiKey.Active {
		return nil, ErrInactiveAPIKey
	}

	// Update last used timestamp
	go a.updateLastUsed(apiKey.ID)

	return &apiKey, nil
}

// updateLastUsed updates the last_used timestamp for an API key
func (a *AuthService) updateLastUsed(id interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"last_used": now}}

	a.collection.UpdateOne(ctx, filter, update)
}

// Close closes the MongoDB connection
func (a *AuthService) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.db.Client().Disconnect(ctx)
}
