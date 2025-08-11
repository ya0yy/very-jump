package config

import (
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	DataDir            string
	Port               string
	JWTSecret          string
	JWTExpiry          time.Duration
	SessionTimeout     time.Duration
	MaxConcurrentConn  int
	RecordingRetention time.Duration
	LogRetention       time.Duration
}

// Load 加载配置
func Load(dataDir string) *Config {
	return &Config{
		DataDir:            dataDir,
		Port:               getEnv("PORT", "8080"),
		JWTSecret:          getEnv("JWT_SECRET", "very-jump-secret-key"),
		JWTExpiry:          getDurationEnv("JWT_EXPIRY", 24*time.Hour),
		SessionTimeout:     getDurationEnv("SESSION_TIMEOUT", 30*time.Minute),
		MaxConcurrentConn:  getIntEnv("MAX_CONCURRENT_CONN", 50),
		RecordingRetention: getDurationEnv("RECORDING_RETENTION", 30*24*time.Hour), // 30 days
		LogRetention:       getDurationEnv("LOG_RETENTION", 90*24*time.Hour),       // 90 days
	}
}

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

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
