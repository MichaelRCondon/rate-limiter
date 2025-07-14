package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the structure of our JWT claims
// This matches the commented structure in main.go
type JWTClaims struct {
	AccountID            int64  `json:"account_id"` // Account ID for rate limiting
	UserID               string `json:"sub"`        // Subject - user identifier
	Role                 string `json:"role"`       // User role (e.g., "admin", "user")
	jwt.RegisteredClaims        // Standard JWT claims (exp, iat, etc.)
}

// Predefined test users for easy token generation
var testUsers = map[string]JWTClaims{
	"user1": {
		AccountID: 12345,
		UserID:    "user123",
		Role:      "user",
	},
	"admin1": {
		AccountID: 99999,
		UserID:    "admin456",
		Role:      "admin",
	},
	"user2": {
		AccountID: 67890,
		UserID:    "user789",
		Role:      "user",
	},
}

func main() {
	// Command line flags
	var (
		secret      = flag.String("secret", "", "JWT secret key (required)")
		userID      = flag.String("user", "", "User ID/subject")
		accountID   = flag.Int64("account", 0, "Account ID for rate limiting")
		role        = flag.String("role", "user", "User role (user, admin)")
		preset      = flag.String("preset", "", "Use predefined user (user1, admin1, user2)")
		duration    = flag.String("duration", "24h", "Token validity duration (e.g., 1h, 24h, 7d)")
		listPresets = flag.Bool("list", false, "List available presets")
		output      = flag.String("output", "token", "Output format: token, header, curl")
	)
	flag.Parse()

	// List presets and exit
	if *listPresets {
		fmt.Println("Available presets:")
		for name, claims := range testUsers {
			fmt.Printf("  %s: AccountID=%d, UserID=%s, Role=%s\n",
				name, claims.AccountID, claims.UserID, claims.Role)
		}
		return
	}

	// Validate required secret
	if *secret == "" {
		log.Fatal("JWT secret is required. Use -secret flag or set JWT_SECRET environment variable")
	}

	// Parse duration
	tokenDuration, err := time.ParseDuration(*duration)
	if err != nil {
		log.Fatalf("Invalid duration format: %v", err)
	}

	var claims JWTClaims

	// Use preset if specified
	if *preset != "" {
		presetClaims, exists := testUsers[*preset]
		if !exists {
			log.Fatalf("Unknown preset '%s'. Use -list to see available presets", *preset)
		}
		claims = presetClaims
	} else {
		// Build claims from individual flags
		if *userID == "" || *accountID == 0 {
			log.Fatal("Either use -preset or provide both -user and -account flags")
		}
		claims = JWTClaims{
			AccountID: *accountID,
			UserID:    *userID,
			Role:      *role,
		}
	}

	// Set standard claims
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenDuration)),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    "rate-limiter-test-tool",
	}

	// Create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(*secret))
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	// Output in requested format
	switch *output {
	case "token":
		fmt.Println(tokenString)
	case "header":
		fmt.Printf("Authorization: Bearer %s\n", tokenString)
	case "curl":
		fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:8080/\n", tokenString)
	case "json":
		result := map[string]interface{}{
			"token":  tokenString,
			"claims": claims,
			"header": fmt.Sprintf("Bearer %s", tokenString),
		}
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonBytes))
	default:
		log.Fatalf("Unknown output format: %s (use: token, header, curl, json)", *output)
	}
}
