package config

import "os"

// Config holds all runtime configuration, sourced from environment variables
// with sensible local-development defaults.
type Config struct {
	HTTPAddr         string // address the HTTP server listens on
	ImageServiceAddr string // gRPC address of the image service
	DatabaseURL      string // Postgres connection string
	TemporalAddress  string // Temporal frontend gRPC address
	TaskQueue        string // Temporal task queue name
	JWTSecret        string // secret key used to sign/verify JWTs (auth service only)
	AuthServiceAddr  string // gRPC address of the auth service
}

// Load reads configuration from the environment, falling back to defaults.
func Load() Config {
	return Config{
		HTTPAddr:         getenv("HTTP_ADDR", ":8080"),
		ImageServiceAddr: getenv("IMAGE_SERVICE_ADDR", "localhost:50051"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://imageservice:secret@localhost:5432/imageservice"),
		TemporalAddress:  getenv("TEMPORAL_ADDRESS", "localhost:7233"),
		TaskQueue:        getenv("TASK_QUEUE", "image-resize"),
		JWTSecret:        getenv("JWT_SECRET", "dev-secret-change-me-in-prod"),
		AuthServiceAddr:  getenv("AUTH_SERVICE_ADDR", "localhost:50052"),
	}
}

// getenv returns the value of the environment variable key, or fallback if it
// is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
