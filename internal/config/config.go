package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTExpiryHours int
	Port          string
}

func Load() (*Config, error) {
	_ = os.Setenv("DATABASE_URL", getEnv("DATABASE_URL", "postgres://localhost:5432/ncvms?sslmode=disable"))
	port := getEnv("PORT", "8080")
	jwtExpiry := 24
	if v := getEnv("JWT_EXPIRY_HOURS", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			jwtExpiry = n
		}
	}
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://localhost:5432/ncvms?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiryHours: jwtExpiry,
		Port:           port,
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
