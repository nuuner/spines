package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabasePath      string
	AdminPassword     string
	GoogleBooksAPIKey string
}

func Load() *Config {
	godotenv.Load() // Load .env file if it exists

	return &Config{
		Port:              getEnv("PORT", "3000"),
		DatabasePath:      getEnv("DATABASE_PATH", "./spines.db"),
		AdminPassword:     getEnv("ADMIN_PASSWORD", ""),
		GoogleBooksAPIKey: getEnv("GOOGLE_BOOKS_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
