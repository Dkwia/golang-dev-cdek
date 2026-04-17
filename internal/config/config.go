package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	JWTSecret       string
	DBHost          string
	DBPort          int
	DBName          string
	DBUser          string
	DBPassword      string
	DBSSLMode       string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("APP_ADDR", ":8080"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		DBHost:          getEnv("DB_HOST", "postgres"),
		DBPort:          getEnvInt("DB_PORT", 5432),
		DBName:          getEnv("DB_NAME", "wishlist"),
		DBUser:          getEnv("DB_USER", "wishlist"),
		DBPassword:      getEnv("DB_PASSWORD", "wishlist"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getEnvDuration("HTTP_IDLE_TIMEOUT", 30*time.Second),
		ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return cfg, nil
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
