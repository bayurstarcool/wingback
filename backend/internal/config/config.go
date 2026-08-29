package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environment-driven settings for the backend.
type Config struct {
	Port        string
	Env         string
	DatabaseURL string
	RedisAddr   string
	RedisPass   string
	JWTSecret   string
	WebBuildDir string
	GeocoderURL string

	DefaultCarrierSpeedKMH float64
	MessageLossProbability float64
	AdSpeedupPercent       float64
	AdSpeedupMaxPerDay     int
}

// Load reads a .env file (if present) then environment variables, applying
// sane defaults for local development.
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://wingback:wingback@localhost:5432/wingback?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:   getEnv("REDIS_PASSWORD", ""),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
		WebBuildDir: getEnv("WEB_BUILD_DIR", ""),
		GeocoderURL: getEnv("GEOCODER_URL", "https://nominatim.openstreetmap.org/reverse"),

		DefaultCarrierSpeedKMH: getEnvFloat("DEFAULT_CARRIER_SPEED_KMH", 177),
		MessageLossProbability: getEnvFloat("MESSAGE_LOSS_PROBABILITY", 0.002),
		AdSpeedupPercent:       getEnvFloat("AD_SPEEDUP_PERCENT", 50),
		AdSpeedupMaxPerDay:     getEnvInt("AD_SPEEDUP_MAX_PER_DAY", 3),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
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
