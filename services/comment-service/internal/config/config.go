package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 保存服务运行配置。
type Config struct {
	Port          string
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	KafkaBrokers  []string
	KafkaTopic    string
	ConsumerGroup string

	ShardCount int
	RateLimit  RateLimitConfig
}

// RateLimitConfig 保存限流配置。
type RateLimitConfig struct {
	WindowSeconds   int
	MaxRequests     int
	IPWindowSeconds int
	IPMaxRequests   int
}

// Load 从环境变量加载配置。
func Load() (Config, error) {
	cfg := Config{
		Port:          getEnv("PORT", "8094"),
		MySQLDSN:      strings.TrimSpace(os.Getenv("MYSQL_DSN")),
		RedisAddr:     getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		KafkaBrokers:  parseCSVEnv("KAFKA_BROKERS", "kafka:9092"),
		KafkaTopic:    getEnv("KAFKA_TOPIC_COMMENT", "comment-events"),
		ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "comment-service-group"),

		ShardCount: getEnvInt("SHARD_COUNT", 8),
		RateLimit: RateLimitConfig{
			WindowSeconds:   getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60),
			MaxRequests:     getEnvInt("RATE_LIMIT_MAX_REQUESTS", 10),
			IPWindowSeconds: getEnvInt("RATE_LIMIT_IP_WINDOW_SECONDS", 60),
			IPMaxRequests:   getEnvInt("RATE_LIMIT_IP_MAX_REQUESTS", 30),
		},
	}

	if cfg.MySQLDSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN is required")
	}
	if cfg.ShardCount <= 0 {
		cfg.ShardCount = 8
	}
	if cfg.RateLimit.WindowSeconds <= 0 {
		cfg.RateLimit.WindowSeconds = 60
	}
	if cfg.RateLimit.MaxRequests <= 0 {
		cfg.RateLimit.MaxRequests = 10
	}
	if cfg.RateLimit.IPWindowSeconds <= 0 {
		cfg.RateLimit.IPWindowSeconds = 60
	}
	if cfg.RateLimit.IPMaxRequests <= 0 {
		cfg.RateLimit.IPMaxRequests = 30
	}
	if len(cfg.KafkaBrokers) == 0 {
		cfg.KafkaBrokers = []string{"kafka:9092"}
	}

	return cfg, nil
}

func parseCSVEnv(key, fallback string) []string {
	raw := getEnv(key, fallback)
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
