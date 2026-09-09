package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort      string
	AutoMigrate  bool
	DefaultLimit int
	MaxLimit     int
	CORSOrigin   string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		AppPort:      env("APP_PORT", "8080"),
		AutoMigrate:  env("AUTO_MIGRATE", "true") == "true",
		DefaultLimit: envInt("DEFAULT_LIMIT", 10),
		MaxLimit:     envInt("MAX_LIMIT", 100),
		CORSOrigin:   env("CORS_ORIGIN", "*"),

		DBHost:     env("DB_HOST", "127.0.0.1"),
		DBPort:     env("DB_PORT", "3306"),
		DBUser:     env("DB_USER", "root"),
		DBPassword: env("DB_PASSWORD", ""),
		DBName:     env("DB_NAME", "article"),
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local&multiStatements=true",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func (c Config) ServerDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true&charset=utf8mb4&loc=Local&multiStatements=true",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
