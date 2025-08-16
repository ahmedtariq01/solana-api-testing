package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// APIKey represents an API key stored in MongoDB
type APIKey struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key       string             `bson:"key" json:"key"`
	Name      string             `bson:"name" json:"name"`
	Active    bool               `bson:"active" json:"active"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	LastUsed  *time.Time         `bson:"last_used,omitempty" json:"last_used,omitempty"`
}
