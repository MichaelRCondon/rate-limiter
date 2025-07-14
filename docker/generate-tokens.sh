#!/bin/bash

# generate-tokens.sh  
# Generates test JWT tokens using docker-compose

set -e

SCRIPT_DIR="$(dirname "$0")"
ENV_FILE="$SCRIPT_DIR/.env"

# Check if .env exists
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: .env file not found. Run generate-env.sh first."
    exit 1
fi

# Source the environment file to get JWT_SECRET
source "$ENV_FILE"

if [ -z "$JWT_SECRET" ]; then
    echo "Error: JWT_SECRET not found in .env file"
    exit 1
fi

echo "JWT Token Generator"
echo "=================="
echo "JWT Secret: [CONFIGURED - ${#JWT_SECRET} characters]"
echo ""

# Build the jwt-signer if needed
echo "Building JWT signer..."
docker-compose build jwt-signer >/dev/null 2>&1

echo "Available presets:"
docker-compose run --rm jwt-signer -list
echo ""

echo "Generating test tokens..."
echo ""

echo "Standard User Tokens:"
echo "User1 (AccountID: 12345):"
USER1_TOKEN=$(docker-compose run --rm jwt-signer -preset=user1)
echo "  Token: $USER1_TOKEN"
echo "  Test:  curl -H \"Authorization: Bearer $USER1_TOKEN\" http://localhost:8080/"
echo ""

echo "User2 (AccountID: 67890):"
USER2_TOKEN=$(docker-compose run --rm jwt-signer -preset=user2)
echo "  Token: $USER2_TOKEN"
echo "  Test:  curl -H \"Authorization: Bearer $USER2_TOKEN\" http://localhost:8080/"
echo ""

echo "Admin User Tokens:"
echo "Admin1 (AccountID: 99999):"
ADMIN1_TOKEN=$(docker-compose run --rm jwt-signer -preset=admin1)
echo "  Token: $ADMIN1_TOKEN"
echo "  Test:  curl -H \"Authorization: Bearer $ADMIN1_TOKEN\" http://localhost:8080/admin/status"
echo ""

# Save tokens to file
TOKENS_FILE="$SCRIPT_DIR/test-tokens.env"
cat > "$TOKENS_FILE" << EOF
# Generated JWT tokens for testing
# Source this file: source test-tokens.env

export USER1_TOKEN="$USER1_TOKEN"
export USER2_TOKEN="$USER2_TOKEN"
export ADMIN1_TOKEN="$ADMIN1_TOKEN"

# Usage examples:
# curl -H "Authorization: Bearer \$USER1_TOKEN" http://localhost:8080/
# curl -H "Authorization: Bearer \$ADMIN1_TOKEN" http://localhost:8080/admin/status
EOF

echo "Tokens saved to: $TOKENS_FILE"
echo ""
echo "Usage:"
echo "  1. Source the tokens: source $TOKENS_FILE"
echo "  2. Test with curl: curl -H \"Authorization: Bearer \$USER1_TOKEN\" http://localhost:8080/"
echo "  3. Generate custom token: docker-compose run jwt-signer -user=custom -account=123 -role=user"