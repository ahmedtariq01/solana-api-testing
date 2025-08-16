// MongoDB initialization script for Docker
// This script runs when the MongoDB container starts for the first time

// Switch to the solana_api database
db = db.getSiblingDB('solana_api');

// Create the api_keys collection
db.createCollection('api_keys');

// Create indexes for optimal performance
db.api_keys.createIndex({ "key": 1 }, { unique: true });
db.api_keys.createIndex({ "active": 1 });
db.api_keys.createIndex({ "key": 1, "active": 1 });

// Insert sample API keys for testing
db.api_keys.insertMany([
  {
    key: "test-api-key-1",
    name: "Test API Key 1",
    active: true,
    created_at: new Date(),
    last_used: null
  },
  {
    key: "test-api-key-2",
    name: "Test API Key 2", 
    active: true,
    created_at: new Date(),
    last_used: null
  },
  {
    key: "inactive-test-key",
    name: "Inactive Test Key",
    active: false,
    created_at: new Date(),
    last_used: null
  },
  {
    key: "docker-dev-key",
    name: "Docker Development Key",
    active: true,
    created_at: new Date(),
    last_used: null
  }
]);

print("Database initialization completed successfully!");
print("Created API keys collection with indexes and sample data.");
print("Available test API keys:");
print("  - test-api-key-1 (active)");
print("  - test-api-key-2 (active)");
print("  - inactive-test-key (inactive)");
print("  - docker-dev-key (active)");