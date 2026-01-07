// Package config provides environment-based configuration management
// Following .rulesgemini Section 7: Load all config from environment variables
package config

import (
	"fmt"
	"os"
	"strconv"
)

// DBConfig holds database connection parameters
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// RedisConfig holds Redis connection parameters
type RedisConfig struct {
	Addr string // Format: host:port
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Port int
}

// FacebookConfig holds Facebook webhook configuration
// Per Phase 2: Webhook verification and HMAC validation
type FacebookConfig struct {
	AppSecret   string // For HMAC SHA256 signature validation
	VerifyToken string // For webhook verification handshake
}

// Config aggregates all configuration sections
type Config struct {
	DB       DBConfig
	Redis    RedisConfig
	App      AppConfig
	Facebook FacebookConfig
}

// LoadConfig reads configuration from environment variables
// Returns error if critical variables are missing
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Database Configuration
	cfg.DB.Host = getEnv("DB_HOST", "chat_os_db")
	cfg.DB.Port = getEnvAsInt("DB_PORT", 3306)
	cfg.DB.User = getEnv("DB_USER", "root")
	cfg.DB.Password = getEnv("DB_PASS", "")
	cfg.DB.Database = getEnv("DB_NAME", "immortal_chat")

	// Validate critical DB password
	if cfg.DB.Password == "" {
		return nil, fmt.Errorf("DB_PASS environment variable is required")
	}

	// Redis Configuration
	cfg.Redis.Addr = getEnv("REDIS_ADDR", "chat_os_redis:6379")

	// Application Configuration
	cfg.App.Port = getEnvAsInt("APP_PORT", 8080)

	// Facebook Configuration (Phase 2)
	cfg.Facebook.AppSecret = getEnv("FB_APP_SECRET", "")
	cfg.Facebook.VerifyToken = getEnv("FB_VERIFY_TOKEN", "")

	// Validate critical Facebook credentials
	if cfg.Facebook.AppSecret == "" {
		return nil, fmt.Errorf("FB_APP_SECRET environment variable is required")
	}
	if cfg.Facebook.VerifyToken == "" {
		return nil, fmt.Errorf("FB_VERIFY_TOKEN environment variable is required")
	}

	return cfg, nil
}

// GetDSN returns MariaDB connection string
func (c *DBConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// getEnv reads environment variable with fallback default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt reads environment variable as integer with fallback default
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
