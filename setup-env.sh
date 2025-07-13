#!/bin/bash

# setup-env.sh - Interactive environment configuration for rate-limiter

set -e

echo "=== Rate Limiter Environment Setup ==="
echo "This script will create a .env file with your configuration."
echo ""

# Clear existing .env file
if [ -f .env ]; then
    echo "Found existing .env file. Creating backup as .env.backup"
    cp .env .env.backup
fi

# Create new .env file
echo "# Rate Limiter Environment Configuration" > .env
echo "# Generated on $(date)" >> .env
echo "" >> .env

# Function to prompt for input with default value
prompt_with_default() {
    local var_name="$1"
    local prompt_text="$2"
    local default_value="$3"
    local is_secret="$4"
    
    if [ "$is_secret" = "true" ]; then
        echo -n "$prompt_text [$default_value]: "
        read -s user_input
        echo ""  # New line after hidden input
    else
        echo -n "$prompt_text [$default_value]: "
        read user_input
    fi
    
    # Use default if input is empty
    if [ -z "$user_input" ]; then
        user_input="$default_value"
    fi
    
    echo "$var_name=$user_input" >> .env
}

# Generate a random password
generate_password() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 32 | tr -d "=+/" | cut -c1-24
    elif command -v head >/dev/null 2>&1 && [ -f /dev/urandom ]; then
        head /dev/urandom | tr -dc A-Za-z0-9 | head -c 24
    else
        echo "temp1234"
    fi
}

# Generate a random JWT secret
generate_jwt_secret() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 48 | tr -d "=+/"
    else
        echo "your-super-secure-jwt-secret-key-that-is-at-least-32-characters-long"
    fi
}

echo "=== Redis Configuration ==="
default_redis_password=$(generate_password)
prompt_with_default "redis_password" "Redis password" "$default_redis_password" "true"

echo ""
echo "=== JWT Configuration ==="
default_jwt_secret=$(generate_jwt_secret)
prompt_with_default "jwt_secret" "JWT secret key" "$default_jwt_secret" "true"

echo ""
echo "=== Database Configuration ==="
prompt_with_default "mongo_url" "MongoDB connection URL" "mongodb://username:password@localhost:27017/ratelimiter" "false"
prompt_with_default "backend_url" "Backend service URL" "http://localhost:3000/api" "false"

echo ""
echo "=== Server Configuration ==="
prompt_with_default "port" "Server port" "8080" "false"
prompt_with_default "default_limit_count" "Default rate limit (requests)" "100" "false"
prompt_with_default "default_period" "Default time period" "1h" "false"

echo ""
echo "Environment configuration complete!"
echo ""
echo "Configuration saved to: .env"
echo "Backup saved to: .env.backup (if .env existed)"
echo ""
echo "Next steps:"
echo "  1. Review the .env file: cat .env"
echo "  2. Start services: docker-compose up -d"
echo "  3. Run the application: go run main.go"
echo ""
