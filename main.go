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
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// Structs

type AuthLevel int

const (
	AuthNone AuthLevel = iota // Ok, iota is slick for sequential generation
	AuthRequired
	AdminRequired
)

type JWTClaims struct {
	AccountID            int64  `json:"account_id"` // Account ID for rate limiting
	UserID               string `json:"sub"`        // Subject - user identifier
	Role                 string `json:"role"`       // User role (e.g., "admin", "user")
	jwt.RegisteredClaims        // Standard JWT claims (exp, iat, etc.)
}

type RateLimitingProxy struct {
	rateLimiter           ratelimiter.RateLimiter
	config                *config.Config
	backendURL            string
	backendHealthcheckURL string
	reverseProxy          httputil.ReverseProxy
}

// Impl

func init() {
	log.Println("DEBUG: init() function called")

	// Load .env file if it exists
	if err := godotenv.Load("./docker/.env"); err != nil {
		log.Printf("DEBUG: godotenv.Load failed: %v", err)
		log.Println("No .env file found, using system environment variables")
	} else {
		log.Println("DEBUG: Successfully loaded .env file")
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
	InfoLogger.Printf("\t\tBackend URL: %s", cfg.BackendConfig.URL)
	InfoLogger.Printf("\t\tServer Port: %d", cfg.ServerConfig.Port)
	InfoLogger.Printf("\t\tDefault Rate Limit: %d requests per %s",
		cfg.DefaultlimitCount, cfg.DefaultPeriod)
	InfoLogger.Printf("\t\tMongoDB URL: %s", sanitizeURL(cfg.MongoURL))
	InfoLogger.Printf("\t\tRedis URL: %s", sanitizeURL(cfg.RedisConfig.URL))
	InfoLogger.Printf("\t\tRatelimiting Algorithm: %s", cfg.LimitingAlgorithm)

	// Print server timeouts
	InfoLogger.Printf("\t\tServer Timeouts - Read: %s, Write: %s, Idle: %s",
		cfg.ServerConfig.ReadTimeout, cfg.ServerConfig.WriteTimeout, cfg.ServerConfig.IdleTimeout)

	// Print JWT info (but not the actual secret)
	if cfg.JWTSecret != "" {
		InfoLogger.Printf("\t\tJWT Secret: [CONFIGURED - %d characters]", len(cfg.JWTSecret))
	} else {
		InfoLogger.Printf("\t\tJWT Secret: [NOT CONFIGURED]")
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
	backendURL, err := url.Parse(cfg.BackendConfig.URL)
	if err != nil {
		return nil, fmt.Errorf("Invalid backend URL %s: %v", cfg.BackendConfig.URL, err)
	}

	revProx := httputil.NewSingleHostReverseProxy(backendURL)

	revProx.ErrorHandler = func(wtr http.ResponseWriter, req *http.Request, err error) {
		ErrorLogger.Printf("Proxy error for %s, %s: %v", req.Method, req.URL.Path, err)
		http.Error(wtr, "Backend Service is not available", http.StatusBadGateway)
	}

	proxy := &RateLimitingProxy{
		rateLimiter:           rateLimiter,
		config:                cfg,
		backendURL:            cfg.BackendConfig.URL,
		backendHealthcheckURL: cfg.BackendConfig.HealthcheckURL,
		reverseProxy:          *revProx,
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
	resp, err := http.Get(prox.backendHealthcheckURL)
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
	// Check configured AuthLevel for the incoming request path
	// Check JWT for appropriate claim, and pull out acctid

	authLevel := prox.determineAuthLevel(req.URL.Path)

	// TODO: STEP 2 - Handle authentication based on the determined level
	var accountId int64
	switch authLevel {
	case AuthNone:
		// Public path - no authentication required - This will be the same as the whitelist path - not limited either
		// Things like healthcheck
		accountId = -1
	case AuthRequired:
		// Standard authentication required
		var err error
		accountId, err = prox.validateJWT(req)
		if err != nil {
			http.Error(wtr, "Unauthorized", http.StatusUnauthorized)
			return
		}
	case AdminRequired:
		// Admin authentication required -
		// Things like reset, add config, etc
		var err error
		accountId, err = prox.validateAdminJWT(req)
		if err != nil {
			http.Error(wtr, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	prox.processRequest(wtr, req, accountId)
}

// TODO: STEP 4 - Move the existing rate limiting and proxy logic into this function
func (prox *RateLimitingProxy) processRequest(wtr http.ResponseWriter, req *http.Request, accountId int64) {
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

func (prox *RateLimitingProxy) determineAuthLevel(path string) AuthLevel {
	// First check if matches any admin paths - this over-rides everything, even if a path is configured for both
	//
	// Next, check if path matches any public paths (no auth needed)

	for _, adminPath := range prox.config.AuthConfig.AdminPaths {
		if prox.pathMatches(path, adminPath) {
			return AdminRequired
		}
	}

	for _, publicPath := range prox.config.AuthConfig.PublicPaths {
		if prox.pathMatches(path, publicPath) {
			return AuthNone
		}

	}
	return AuthRequired
}

func (prox *RateLimitingProxy) pathMatches(requestPath, configPath string) bool {
	// Exact match
	if requestPath == configPath {
		return true
	}

	// Wildcard match (e.g., "/admin/*" matches "/admin/users")
	if strings.HasSuffix(configPath, "/*") {
		prefix := strings.TrimSuffix(configPath, "/*")
		return strings.HasPrefix(requestPath, prefix)
	}

	return false
}

func (prox *RateLimitingProxy) getJWTFromHeader(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing required Authorization header")
	}

	// Check for "Bearer " prefix
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	// Extract token (remove "Bearer " prefix)
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("missing required JWT")
	}

	return token, nil
}

func (prox *RateLimitingProxy) parseJWT(tokenString string) (*JWTClaims, error) {
	// Parse token with claims
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(prox.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	// Pull out the claims
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid JWT claims")
}

func (prox *RateLimitingProxy) validateJWT(req *http.Request) (int64, error) {
	// Extract token from header
	tokenString, err := prox.getJWTFromHeader(req)
	if err != nil {
		return 0, err
	}

	// Parse and validate token
	claims, err := prox.parseJWT(tokenString)
	if err != nil {
		return 0, err
	}

	// Validate account ID is present
	if claims.AccountID <= 0 {
		return 0, fmt.Errorf("invalid account ID in JWT")
	}

	return claims.AccountID, nil
}

func (prox *RateLimitingProxy) validateAdminJWT(req *http.Request) (int64, error) {
	// Extract token from header
	tokenString, err := prox.getJWTFromHeader(req)
	if err != nil {
		return 0, err
	}

	// Parse and validate token
	claims, err := prox.parseJWT(tokenString)
	if err != nil {
		return 0, err
	}

	// Validate account ID is present
	if claims.AccountID <= 0 {
		return 0, fmt.Errorf("invalid account ID in JWT")
	}

	// Check admin role
	if claims.Role != "admin" {
		return 0, fmt.Errorf("insufficient privileges: admin role required")
	}

	return claims.AccountID, nil
}

// setupGracefulShutdown handles SIGINT/SIGTERM for clean shutdown
func setupGracefulShutdown() {
	InfoLogger.Println("Setting up graceful shutdown...")

}

// sanitizeURL removes credentials from URLs for safe logging
func sanitizeURL(url string) string {
	// TODO: Remove username/password from URLs before logging
	return url
}
