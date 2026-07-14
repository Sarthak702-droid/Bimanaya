package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	RedisURL           string
	S3Endpoint         string
	S3AccessKey        string
	S3SecretKey        string
	S3BucketName       string
	AIWorkerURL        string
	ClerkSecretKey     string
	ClerkJWTIssuer     string
	ClerkJWKSURL       string
	ConvexURL          string
	ConvexDeployKey    string
	CORSAllowedOrigins string
	Environment        string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		RedisURL:           getEnv("REDIS_URL", "localhost:6379"),
		S3Endpoint:         getEnv("S3_ENDPOINT", "http://localhost:9000"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        getEnv("S3_SECRET_KEY", "minioadmin"),
		S3BucketName:       getEnv("S3_BUCKET_NAME", "bimanyaya-docs"),
		AIWorkerURL:        getEnv("AI_WORKER_URL", "http://localhost:8000"),
		ClerkSecretKey:     getEnv("CLERK_SECRET_KEY", ""),
		ClerkJWTIssuer:     getEnv("CLERK_JWT_ISSUER", "https://alert-ghost-7.clerk.accounts.dev"),
		ClerkJWKSURL:       getEnv("CLERK_JWKS_URL", "https://alert-ghost-7.clerk.accounts.dev/.well-known/jwks.json"),
		ConvexURL:          getEnv("CONVEX_URL", "https://alert-ghost-7.convex.cloud"),
		ConvexDeployKey:    getEnv("CONVEX_DEPLOY_KEY", ""),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		Environment:        getEnv("ENV", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}
