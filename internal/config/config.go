package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port            string
	SupabaseURL     string
	SupabaseAnonKey string
	MongoDBURI      string
	MongoDBPassword string
	Environment     string
	LogLevel        string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:            getEnvWithDefault("PORT", "8080"),
		SupabaseURL:     os.Getenv("SUPABASE_URL"),
		SupabaseAnonKey: os.Getenv("SUPABASE_URL_ANON_KEY"),
		MongoDBURI:      os.Getenv("MONGODB_URI"),
		MongoDBPassword: os.Getenv("MONGODB_PASSWORD"),
		Environment:     getEnvWithDefault("ENVIRONMENT", "development"),
		LogLevel:        getEnvWithDefault("LOG_LEVEL", "info"),
	}

	// Validate required fields
	if cfg.SupabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL is required")
	}
	if cfg.SupabaseAnonKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL_ANON_KEY is required")
	}
	if cfg.MongoDBURI == "" {
		return nil, fmt.Errorf("MONGODB_URI is required")
	}
	if cfg.MongoDBPassword == "" {
		return nil, fmt.Errorf("MONGODB_PASSWORD is required")
	}

	return cfg, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}
