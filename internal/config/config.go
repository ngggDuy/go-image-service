package config

import (
	"os"
	"strconv"
)

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

	MinioEndpoint       string // internal endpoint for object operations (e.g. minio:9000)
	MinioPublicEndpoint string // client-reachable endpoint for presigned URLs (e.g. localhost:9000)
	MinioAccessKey      string
	MinioSecretKey      string
	MinioBucket         string

	TemporalNamespace string // Temporal Cloud namespace, as <namespace_id>.<account_id>
	TemporalAPIKey    string // Temporal Cloud API key; empty means a local, unauthenticated cluster

	ObjectStoreUseSSL       bool // https to the object store (true for GCS, false for local MinIO)
	ObjectStoreEnsureBucket bool // create the bucket at startup; false against GCS, which rejects it
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

		MinioEndpoint:       getenv("MINIO_ENDPOINT", "localhost:9000"),
		MinioPublicEndpoint: getenv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinioAccessKey:      getenv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:      getenv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:         getenv("MINIO_BUCKET", "images"),

		TemporalNamespace: getenv("TEMPORAL_NAMESPACE", "default"),
		TemporalAPIKey:    getenv("TEMPORAL_API_KEY", ""),

		ObjectStoreUseSSL:       getenvBool("OBJECT_STORE_USE_SSL", false),
		ObjectStoreEnsureBucket: getenvBool("OBJECT_STORE_ENSURE_BUCKET", true),
	}
}

// ListenAddr returns the TCP address to listen on. Cloud Run injects PORT and
// requires the container to listen on it; locally PORT is unset and we fall
// back to the service's conventional port.
func ListenAddr(fallback string) string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":" + fallback
}

// getenv returns the value of the environment variable key, or fallback if it
// is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getenvBool returns the boolean value of the environment variable key, or
// fallback if it is unset, empty, or not parseable as a bool.
func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
