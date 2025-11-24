package configuration

import (
	"fmt"
	"os"
)

type Config struct {
	Database    DatabaseConfig
	MinIO       MinIOConfig
	Server      ServerConfig
	NATSURL     string
	KeycloakUrl string
	CLAMAVURL   string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type MinIOConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	UseSSL     bool
}

type ServerConfig struct {
	Port string
}

func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "fileuser"),
			Password: getEnv("DB_PASSWORD", "filepassword"),
			DBName:   getEnv("DB_NAME", "filemanager"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		MinIO: MinIOConfig{
			Endpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
			BucketName: getEnv("MINIO_BUCKET", "files"),
			UseSSL:     getEnv("MINIO_USE_SSL", "false") == "true",
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		NATSURL:     getEnv("NATS_URL", "nats://localhost:4222"),
		CLAMAVURL:   getEnv("CLAMAV_URL", "tcp://localhost:3310"),
		KeycloakUrl: getEnv("KEYCLOAK_URL", "http://localhost:8081/realms/bondbridg"),
	}
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
