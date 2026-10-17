package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
)

type Config struct {
	HTTPPort   string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
}

// getEnv возвращает значение переменной
// или значение по умолчанию.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// Load считывает настройки из переменных окружения.
func Load() (Config, error) {
	cfg := Config{
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     os.Getenv("DB_NAME"),
		DBUser:     os.Getenv("APP_DB_USER"),
		DBPassword: os.Getenv("APP_DB_PASSWORD"),
	}

	if cfg.DBName == "" ||
		cfg.DBUser == "" ||
		cfg.DBPassword == "" {
		return Config{}, fmt.Errorf(
			"required database environment variables are missing in config",
		)
	}

	return cfg, nil
}

// DatabaseURL формирует строку подключения к PostgreSQL.
func (c Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   net.JoinHostPort(c.DBHost, c.DBPort),
		Path:   "/" + c.DBName,
	}

	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()

	return u.String()
}
