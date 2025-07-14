#!/bin/bash

# start-local.sh - One-command local startup for rate-limiter (Linux/macOS)

set -e

echo "=== Rate Limiter Local Startup ==="

# Check if .env already exists
if [ -f docker/.env ]; then
    echo "Found existing docker/.env file - preserving credentials to maintain Redis data access"
    echo ""
    read -p "Regenerate new credentials? This will lose existing Redis data [y/N]: " regenerate
    if [[ ! "$regenerate" =~ ^[Yy]$ ]]; then
        echo "Using existing credentials..."
        echo ""
    else
        echo "Creating backup of existing .env..."
        cp docker/.env docker/.env.backup
        echo ""
        generate_new=true
    fi
else
    generate_new=true
fi

if [ "$generate_new" = true ]; then
    echo "Generating new secure credentials..."
    echo ""

    # Function to generate secure random password
    generate_password() {
        local length="$1"
        if command -v openssl >/dev/null 2>&1; then
            openssl rand -base64 $((length * 3 / 4)) | tr -d "=+/" | head -c "$length"
        elif [ -f /dev/urandom ]; then
            head /dev/urandom | tr -dc 'A-Za-z0-9' | head -c "$length"
        else
            # Fallback for systems without openssl or /dev/urandom
            date +%s | sha256sum | base64 | head -c "$length"
        fi
    }

    # Generate secure credentials
    redis_password=$(generate_password 24)
    jwt_secret=$(generate_password 48)

    # Create .env file in docker directory
    cat > docker/.env << EOF
# Rate Limiter Environment Configuration
# Generated on $(date)

# Redis Configuration
redis_password=$redis_password

# JWT Configuration
jwt_secret=$jwt_secret
EOF

    echo "New credentials generated and saved to docker/.env"
    echo "WARNING: New Redis password means existing Redis data will be inaccessible"
    echo ""
fi

# Start Docker services
echo "Starting Docker services..."
cd docker

if docker-compose up -d; then
    echo ""
    echo "=== Services Started Successfully! ==="
    echo ""
    echo "Rate Limiter: http://localhost:8080"
    echo "Demo API: http://localhost:9080/hello"
    echo "Redis Commander: http://localhost:8081"
    echo ""
    echo "To stop services: docker-compose down"
    echo "To view logs: docker-compose logs -f"
else
    echo ""
    echo "ERROR: Failed to start services"
    echo "Check Docker installation and try again"
    exit 1
fi

cd ..