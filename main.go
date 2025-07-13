package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"rate-limiter/config"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func init() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env doesn't exist (production might use real env vars)
		log.Println("No .env file found, using system environment variables")
	}
}

// Application-wide logger
var (
	InfoLogger  = log.New(os.Stdout, "[MAIN] INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "[MAIN] ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func main() {
	InfoLogger.Println("Starting rate-limiter proxy...")

	// Step 1: Load and validate configuration
	cfg, err := loadConfig()
	if err != nil {
		ErrorLogger.Fatal("Failed to load configuration:", err)
	}

	// Step 2: Print config summary for debugging
	printConfigSummary(cfg)

	err = initializeStorage(cfg)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	// Step 4: Start HTTP server (future)
	// err = startServer(cfg)
	// if err != nil {
	//     ErrorLogger.Fatal("Failed to start server:", err)
	// }

	InfoLogger.Println("Rate-limiter proxy started successfully")

	// Step 5: Handle graceful shutdown (future)
	// setupGracefulShutdown()
}

// loadConfig loads configuration from file/env and validates it
func loadConfig() (*config.Config, error) {
	InfoLogger.Println("Loading configuration...")

	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	InfoLogger.Println("Configuration loaded and validated successfully")
	return cfg, nil
}

// printConfigSummary prints a summary of loaded config for debugging
func printConfigSummary(cfg *config.Config) {
	InfoLogger.Println("#### Configuration Summary ####")

	// Print key config values (but NOT secrets like JWT)
	InfoLogger.Printf("Backend URL: %s", cfg.BackendURL)
	InfoLogger.Printf("Server Port: %d", cfg.ServerConfig.Port)
	InfoLogger.Printf("Default Rate Limit: %d requests per %s",
		cfg.DefaultlimitCount, cfg.DefaultPeriod)
	InfoLogger.Printf("MongoDB URL: %s", sanitizeURL(cfg.MongoURL))
	InfoLogger.Printf("Redis URL: %s", sanitizeURL(cfg.RedisConfig.URL))

	// Print server timeouts
	InfoLogger.Printf("Server Timeouts - Read: %s, Write: %s, Idle: %s",
		cfg.ServerConfig.ReadTimeout, cfg.ServerConfig.WriteTimeout, cfg.ServerConfig.IdleTimeout)

	// Print JWT info (but not the actual secret)
	if cfg.JWTSecret != "" {
		InfoLogger.Printf("JWT Secret: [CONFIGURED - %d characters]", len(cfg.JWTSecret))
	} else {
		InfoLogger.Printf("JWT Secret: [NOT CONFIGURED]")
	}

	InfoLogger.Println("#### End Configuration Summary ####")
}

// initializeStorage sets up MongoDB and Redis connections
func initializeStorage(cfg *config.Config) error {
	InfoLogger.Println("Initializing storage connections...")

	// TODO: Initialize MongoDB connection
	// mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURL))
	// if err != nil {
	//     return fmt.Errorf("failed to connect to MongoDB: %w", err)
	// }
	var redisClient *redis.Client

	if cfg.RedisConfig.Username != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisConfig.URL,
			Password: cfg.RedisConfig.Password,
			DB:       cfg.RedisConfig.DB,
			Username: cfg.RedisConfig.Username,
		})
	} else {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisConfig.URL,
			Password: cfg.RedisConfig.Password,
			DB:       cfg.RedisConfig.DB,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Unable to reach Redis service at %s : %s", sanitizeURL(cfg.RedisConfig.URL), err)
	}

	InfoLogger.Println("Storage connections initialized successfully")
	return nil
}

// startServer starts the HTTP proxy server
func startServer(cfg *config.Config) error {
	InfoLogger.Printf("Starting HTTP server on port %d...", cfg.ServerConfig.Port)

	// TODO: Set up HTTP handlers
	// TODO: Configure server with timeouts from config
	// TODO: Start listening

	return nil
}

// setupGracefulShutdown handles SIGINT/SIGTERM for clean shutdown
func setupGracefulShutdown() {
	InfoLogger.Println("Setting up graceful shutdown...")

	// TODO: Set up signal handling
	// TODO: Gracefully close database connections
	// TODO: Gracefully shutdown HTTP server
}

// sanitizeURL removes credentials from URLs for safe logging
func sanitizeURL(url string) string {
	// TODO: Remove username/password from URLs before logging
	return url
}
