package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig    `json:"server"`
	MongoDB   MongoDBConfig   `json:"mongodb"`
	RPC       RPCConfig       `json:"rpc"`
	Cache     CacheConfig     `json:"cache"`
	RateLimit RateLimitConfig `json:"rate_limit"`
	Logging   LoggingConfig   `json:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string        `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// MongoDBConfig holds MongoDB connection configuration
type MongoDBConfig struct {
	URI              string        `json:"uri"`
	Database         string        `json:"database"`
	APIKeyCollection string        `json:"api_key_collection"`
	ConnectTimeout   time.Duration `json:"connect_timeout"`
	MaxPoolSize      uint64        `json:"max_pool_size"`
}

// RPCConfig holds Solana RPC configuration
type RPCConfig struct {
	Endpoint           string        `json:"endpoint"`
	Timeout            time.Duration `json:"timeout"`
	APIKey             string        `json:"api_key"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	ConnectionPoolSize int           `json:"connection_pool_size"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	TTL             time.Duration `json:"ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxSize         int           `json:"max_size"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	WindowSize        time.Duration `json:"window_size"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level       string   `json:"level"`
	Environment string   `json:"environment"`
	OutputPaths []string `json:"output_paths"`
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		MongoDB: MongoDBConfig{
			URI:              getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:         getEnv("MONGODB_DATABASE", "solana_api"),
			APIKeyCollection: getEnv("MONGODB_APIKEY_COLLECTION", "api_keys"),
			ConnectTimeout:   getDurationEnv("MONGODB_CONNECT_TIMEOUT", 10*time.Second),
			MaxPoolSize:      getUint64Env("MONGODB_MAX_POOL_SIZE", 100),
		},
		RPC: RPCConfig{
			Endpoint:           getEnv("SOLANA_RPC_ENDPOINT", "https://pomaded-lithotomies-xfbhnqagbt-dedicated.helius-rpc.com/?api-key=37ba4475-8fa3-4491-875f-758894981943"),
			Timeout:            getDurationEnv("SOLANA_RPC_TIMEOUT", 30*time.Second),
			APIKey:             getEnv("SOLANA_RPC_API_KEY", "37ba4475-8fa3-4491-875f-758894981943"),
			MaxRetries:         getIntEnv("SOLANA_RPC_MAX_RETRIES", 3),
			RetryDelay:         getDurationEnv("SOLANA_RPC_RETRY_DELAY", 1*time.Second),
			ConnectionPoolSize: getIntEnv("SOLANA_RPC_CONNECTION_POOL_SIZE", 10),
		},
		Cache: CacheConfig{
			TTL:             getDurationEnv("CACHE_TTL", 10*time.Second),
			CleanupInterval: getDurationEnv("CACHE_CLEANUP_INTERVAL", 60*time.Second),
			MaxSize:         getIntEnv("CACHE_MAX_SIZE", 10000),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getIntEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", 10),
			WindowSize:        getDurationEnv("RATE_LIMIT_WINDOW_SIZE", time.Minute),
			CleanupInterval:   getDurationEnv("RATE_LIMIT_CLEANUP_INTERVAL", 5*time.Minute),
		},
		Logging: LoggingConfig{
			Level:       getEnv("LOG_LEVEL", "info"),
			Environment: getEnv("LOG_ENVIRONMENT", "development"),
			OutputPaths: getStringSliceEnv("LOG_OUTPUT_PATHS", []string{"stdout"}),
		},
	}
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getUint64Env(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if uint64Value, err := strconv.ParseUint(value, 10, 64); err == nil {
			return uint64Value
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		return []string{value}
	}
	return defaultValue
}
