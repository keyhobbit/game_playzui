package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppPort    int
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	RedisAddr  string
	RedisPwd   string
	JWTSecret  string
}

func Load() *Config {
	return &Config{
		AppPort:    getEnvInt("APP_PORT", 8700),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "tienlen"),
		DBPassword: getEnv("DB_PASSWORD", "tienlen_secret"),
		DBName:     getEnv("DB_NAME", "tienlen"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPwd:   getEnv("REDIS_PASSWORD", ""),
		JWTSecret:  getEnv("JWT_SECRET", "dev-secret-key"),
	}
}

func (c *Config) DSN() string {
	return "host=" + c.DBHost +
		" port=" + strconv.Itoa(c.DBPort) +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
