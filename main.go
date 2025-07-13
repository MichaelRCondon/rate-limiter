package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"rate-limiter/config"
	"rate-limiter/ratelimiter"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// Structs
type Proxy struct {
	rateLimiter ratelimiter.RateLimiter
	config      *config.Config
	backendURL  string
}

// Impl

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

	cfg, err := loadConfig()
	if err != nil {
		ErrorLogger.Fatal("Failed to load configuration:", err)
	}

	printConfigSummary(cfg)

	client, err := initializeStorage(cfg)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	err = startServer(cfg, client)
	if err != nil {
		ErrorLogger.Fatal("Unable to start server", err)
	}

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
func initializeStorage(cfg *config.Config) (*redis.Client, error) {
	InfoLogger.Println("Initializing storage connections...")

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
		return nil, fmt.Errorf("Unable to reach Redis service at %s : %s", sanitizeURL(cfg.RedisConfig.URL), err)
	}

	InfoLogger.Println("Storage connections initialized successfully")
	return redisClient, nil
}

// startServer starts the HTTP proxy server
func startServer(cfg *config.Config, redClient *redis.Client) error {
	InfoLogger.Printf("Starting HTTP server on port %d...", cfg.ServerConfig.Port)
	rateLimiter, err := ratelimiter.NewRateLimiter("allow_all", redClient)

	if err != nil {
		ErrorLogger.Fatalf("Unable to load RateLimiter", err)
	}

	proxy := &Proxy{
		rateLimiter: rateLimiter,
		config:      cfg,
		backendURL:  cfg.BackendURL,
	}

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.ServerConfig.Port),
		Handler:      http.HandlerFunc(proxy.handleRequest),
		ReadTimeout:  cfg.ServerConfig.ReadTimeout,
		IdleTimeout:  cfg.ServerConfig.IdleTimeout,
		WriteTimeout: cfg.ServerConfig.WriteTimeout,
	}

	InfoLogger.Printf("HTTP server listening on port %d", cfg.ServerConfig.Port)
	return server.ListenAndServe() // This blocks until server stops
}

func (prox *Proxy) handleRequest(wtr http.ResponseWriter, req *http.Request) {
	// Pull/check JWT, grab acctid
	// Call the rate limiter
	//		if allowed - forward
	//		if not, return 429

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
