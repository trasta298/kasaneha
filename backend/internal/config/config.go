package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	AI       AIConfig
	JWT      JWTConfig
	Redis    RedisConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxConnections  int
	MinConnections  int
	MaxConnLifetime string
	MaxConnIdleTime string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string
	Port string
	Env  string
}

// AIConfig holds AI service configuration
type AIConfig struct {
	GeminiAPIKey string
	Model        string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	cfg := &Config{
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://kasaneha:password@localhost:5432/kasaneha_db?sslmode=disable"),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 30),
			MinConnections:  getEnvAsInt("DB_MIN_CONNECTIONS", 5),
			MaxConnLifetime: getEnv("DB_MAX_CONN_LIFETIME", "1h"),
			MaxConnIdleTime: getEnv("DB_MAX_CONN_IDLE_TIME", "30m"),
		},
		Server: ServerConfig{
			Host: getEnv("HOST", "0.0.0.0"),
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		AI: AIConfig{
			GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
			Model:        getEnv("GEMINI_MODEL", "gemini-2.5-flash-preview-05-20"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-secret-key"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
	}

	return cfg, nil
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Server.Env) == "development"
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Server.Env) == "production"
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvAsInt gets an environment variable as integer with a fallback value
func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
