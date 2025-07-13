package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"rate-limiter/config"
	"rate-limiter/ratelimiter"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// Structs
type RateLimitingProxy struct {
	rateLimiter  ratelimiter.RateLimiter
	config       *config.Config
	backendURL   string
	reverseProxy httputil.ReverseProxy
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

func setupProxy(cfg *config.Config, rateLimiter ratelimiter.RateLimiter) (*RateLimitingProxy, error) {
	// Set up the backend URL
	// -- TODO FUTURE -- Extend this with a ChooseBackend function based on inbound host
	backendURL, err := url.Parse(cfg.BackendURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid backend URL %s: %v", cfg.BackendURL, err)
	}

	revProx := httputil.NewSingleHostReverseProxy(backendURL)

	revProx.ErrorHandler = func(wtr http.ResponseWriter, req *http.Request, err error) {
		ErrorLogger.Printf("Proxy error for %s, %s: %v", req.Method, req.URL.Path, err)
		http.Error(wtr, "Backend Service is not available", http.StatusBadGateway)
	}

	proxy := &RateLimitingProxy{
		rateLimiter:  rateLimiter,
		config:       cfg,
		backendURL:   cfg.BackendURL,
		reverseProxy: *revProx,
	}
	return proxy, nil
}

// startServer starts the HTTP proxy server
func startServer(cfg *config.Config, redClient *redis.Client) error {
	InfoLogger.Printf("Starting HTTP server on port %d...", cfg.ServerConfig.Port)
	rateLimiter, err := ratelimiter.NewRateLimiter(cfg.LimitingAlgorithm, redClient, cfg.DefaultPeriod, cfg.DefaultlimitCount)
	if err != nil {
		ErrorLogger.Fatalf("Unable to load RateLimiter", err)
	}

	proxy, err := setupProxy(cfg, rateLimiter)
	if err != nil {
		ErrorLogger.Fatalf("Unable to set up reverse proxy", err)
	}

	// We need muxing to trap *all* requests
	mux := http.NewServeMux()
	mux.HandleFunc("/health", proxy.handleHealth) // We're going to have a simple health endpoint for kube
	mux.HandleFunc("/", proxy.handleRequest)      // Everything else is rate-limited

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.ServerConfig.Port),
		Handler:      mux,
		ReadTimeout:  cfg.ServerConfig.ReadTimeout,
		IdleTimeout:  cfg.ServerConfig.IdleTimeout,
		WriteTimeout: cfg.ServerConfig.WriteTimeout,
	}

	InfoLogger.Printf("HTTP server listening on port %d", cfg.ServerConfig.Port)
	return server.ListenAndServe() // This blocks until server stops
}

func (prox *RateLimitingProxy) handleHealth(wtr http.ResponseWriter, req *http.Request) {
	// FUTURE TODO - the back-end really should also have a healthcheck
	resp, err := http.Get(prox.backendURL + "/hello")
	if err != nil {
		ErrorLogger.Printf("Backend health check failed: %v", err)
		http.Error(wtr, "Backend unhealthy", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ErrorLogger.Printf("Backend returned status %d", resp.StatusCode)
		http.Error(wtr, "Backend unhealthy", http.StatusServiceUnavailable)
		return
	}

	wtr.Header().Set("Content-Type", "application/json")
	wtr.WriteHeader(http.StatusOK)
	wtr.Write([]byte(`{"status": "healthy", "backend": "reachable"}`))
}

func (prox *RateLimitingProxy) handleRequest(wtr http.ResponseWriter, req *http.Request) {
	// Pull/check JWT, grab acctid
	var accountId int64
	accountId = -1 // For testing

	// Call the rate limiter
	//		if allowed - forward
	//		if not, return 429
	ctx, cancel := context.WithTimeout(req.Context(), 600*time.Second)
	defer cancel()

	result, err := prox.rateLimiter.CheckLimit(ctx, accountId, req.URL.Path)

	// Fail closed
	if err != nil { // If the check fails, fail closed
		ErrorLogger.Printf("RateLimit check failed - AccountID: %d, %v", accountId, err)
		http.Error(wtr, "Rate Limiting Unavailable", http.StatusInternalServerError)
		return
	}

	// Add proxy headers for limit/remaining
	if result.Limit > 0 {
		wtr.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
	}

	if result.Remaining >= 0 {
		wtr.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
	}

	// If the limiter says no...
	if !result.Allowed {
		InfoLogger.Printf("Rate limit exceeded for account %d on path %s", accountId, req.URL.Path)

		if result.RetryAfter >= 0 {
			wtr.Header().Set("Retry-After", fmt.Sprintf("%.0f", result.RetryAfter.Seconds()))
		}
		http.Error(wtr, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	InfoLogger.Printf("Proxying request to backend - AccountID: %d, %s, %s", accountId, req.Method, req.URL.Path)

	originalDirector := prox.reverseProxy.Director
	prox.reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-By", "rate-limiter-proxy")
		req.Header.Set("X-Proxy-Version", "1.0")
		req.Header.Set("X-Account-ID", fmt.Sprintf("%d", accountId))
		InfoLogger.Printf("Forwarding %s %s to %s", req.Method, req.URL.Path, req.URL.String())
	}

	prox.reverseProxy.ServeHTTP(wtr, req)
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
