package config

/*import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the rate limiter service
type Config struct {
	Server    ServerConfig    `json:"server"`
	Kafka     KafkaConfig     `json:"kafka"`
	RateLimit RateLimitConfig `json:"rate_limit"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string        `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// KafkaConfig holds Kafka connection and topic configuration
type KafkaConfig struct {
	Brokers     []string    `json:"brokers"`
	Topics      TopicConfig `json:"topics"`
	GroupID     string      `json:"group_id"`
	RetryBuffer RetryConfig `json:"retry_buffer"`
}

// TopicConfig defines all Kafka topics used by the service
type TopicConfig struct {
	Requests  string `json:"requests"`
	State     string `json:"state"`
	Decisions string `json:"decisions"`
	Retry     string `json:"retry"`
}

// RetryConfig holds retry buffer configuration
type RetryConfig struct {
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`
	BufferTTL  time.Duration `json:"buffer_ttl"`
	BatchSize  int           `json:"batch_size"`
}

// RateLimitConfig holds rate limiting rules and algorithms
type RateLimitConfig struct {
	DefaultRules RateLimitRules `json:"default_rules"`
	Algorithm    string         `json:"algorithm"` // "token_bucket", "sliding_window"
}

// RateLimitRules defines rate limiting parameters
type RateLimitRules struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	WindowSize        time.Duration `json:"window_size"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Kafka: KafkaConfig{
			Brokers: getSliceEnv("KAFKA_BROKERS", []string{"localhost:9092"}),
			GroupID: getEnv("KAFKA_GROUP_ID", "rate-limiter"),
			Topics: TopicConfig{
				Requests:  getEnv("KAFKA_TOPIC_REQUESTS", "rate-limit-requests"),
				State:     getEnv("KAFKA_TOPIC_STATE", "rate-limit-state"),
				Decisions: getEnv("KAFKA_TOPIC_DECISIONS", "rate-limit-decisions"),
				Retry:     getEnv("KAFKA_TOPIC_RETRY", "rate-limit-retry"),
			},
			RetryBuffer: RetryConfig{
				MaxRetries: getIntEnv("RETRY_MAX_RETRIES", 3),
				RetryDelay: getDurationEnv("RETRY_DELAY", 30*time.Second),
				BufferTTL:  getDurationEnv("RETRY_BUFFER_TTL", 5*time.Minute),
				BatchSize:  getIntEnv("RETRY_BATCH_SIZE", 10),
			},
		},
		RateLimit: RateLimitConfig{
			Algorithm: getEnv("RATE_LIMIT_ALGORITHM", "token_bucket"),
			DefaultRules: RateLimitRules{
				RequestsPerSecond: getIntEnv("RATE_LIMIT_RPS", 10),
				BurstSize:         getIntEnv("RATE_LIMIT_BURST", 20),
				WindowSize:        getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
			},
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka brokers cannot be empty")
	}

	if c.RateLimit.DefaultRules.RequestsPerSecond <= 0 {
		return fmt.Errorf("requests per second must be positive")
	}

	if c.RateLimit.DefaultRules.BurstSize <= 0 {
		return fmt.Errorf("burst size must be positive")
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple split by comma - in production you might want more sophisticated parsing
		return []string{value} // For now, just return single value
	}
	return defaultValue
}*/
