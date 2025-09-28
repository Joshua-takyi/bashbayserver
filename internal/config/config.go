package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port                string
	SupabaseURL         string
	SupabaseAnonKey     string
	MongoDBURI          string
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	MongoDBPassword     string
	Environment         string
	LogLevel            string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:                getEnvWithDefault("PORT", "8080"),
		SupabaseURL:         os.Getenv("SUPABASE_URL"),
		SupabaseAnonKey:     os.Getenv("SUPABASE_URL_ANON_KEY"),
		MongoDBURI:          os.Getenv("MONGODB_URI"),
		CloudinaryCloudName: os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey:    os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret: os.Getenv("CLOUDINARY_API_SECRET"),
		MongoDBPassword:     os.Getenv("MONGODB_PASSWORD"),
		Environment:         getEnvWithDefault("ENVIRONMENT", "development"),

		LogLevel: getEnvWithDefault("LOG_LEVEL", "info"),
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
	if cfg.CloudinaryCloudName == "" {
		return nil, fmt.Errorf("CLODINARY_CLOUD_NAME is required")
	}
	if cfg.CloudinaryAPIKey == "" {
		return nil, fmt.Errorf("CLODINARY_API_KEY is required")
	}
	if cfg.CloudinaryAPISecret == "" {
		return nil, fmt.Errorf("CLODINARY_API_SECRET is required")
	}

	// if
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
