package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBURL         string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	ServerPort    string
}

func Load() (*Config, error) {
	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "1h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5434")
	user := getEnv("DB_USER", "familycotton")
	password := getEnv("DB_PASSWORD", "familycotton_secret")
	dbName := getEnv("DB_NAME", "familycotton")

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbName)

	return &Config{
		DBHost:        host,
		DBPort:        port,
		DBUser:        user,
		DBPassword:    password,
		DBName:        dbName,
		DBURL:         dbURL,
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		JWTAccessTTL:  accessTTL,
		JWTRefreshTTL: refreshTTL,
		ServerPort:    getEnv("SERVER_PORT", "8082"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
