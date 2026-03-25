package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL             string
	JWTSecret               string
	JWTExpiryHours          int
	Port                    string
	ChildLinkOTPTTLMin      int
	ChildLinkOTPCooldownSec int
	ChildLinkOTPMaxAttempts int
}

func Load() (*Config, error) {
	_ = os.Setenv("DATABASE_URL", getEnv("DATABASE_URL", "postgres://localhost:5432/ncvms?sslmode=disable"))
	port := getEnv("PORT", "8080")
	jwtExpiry := getIntEnv("JWT_EXPIRY_HOURS", 24)

	return &Config{
		DatabaseURL:             getEnv("DATABASE_URL", "postgres://localhost:5432/ncvms?sslmode=disable"),
		JWTSecret:               getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiryHours:          jwtExpiry,
		Port:                    port,
		ChildLinkOTPTTLMin:      getIntEnv("CHILD_LINK_OTP_TTL_MIN", 5),
		ChildLinkOTPCooldownSec: getIntEnv("CHILD_LINK_OTP_COOLDOWN_SEC", 60),
		ChildLinkOTPMaxAttempts: getIntEnv("CHILD_LINK_OTP_MAX_ATTEMPTS", 5),
	}, nil
}

func getIntEnv(key string, def int) int {
	if v := getEnv(key, ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
