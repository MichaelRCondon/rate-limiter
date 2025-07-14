#!/bin/bash

# test-examples.sh
# Examples of generating different JWT tokens

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <jwt-secret>"
    echo ""
    echo "Examples:"
    echo "  $0 'my-secret-key'"
    exit 1
fi

JWT_SECRET="$1"

echo "JWT Token Test Examples"
echo "======================"
echo ""

# Build if needed
if [ ! -f "./jwt-signer" ]; then
    echo "Building jwt-signer..."
    go build -o jwt-signer main.go
fi

echo "1. List available presets:"
./jwt-signer -list
echo ""

echo "2. Generate tokens for each preset:"
echo ""

echo "Standard User 1:"
USER1_TOKEN=$(./jwt-signer -secret="$JWT_SECRET" -preset=user1)
echo "  AccountID: 12345, Role: user"
echo "  Token: $USER1_TOKEN"
echo "  Test: curl -H \"Authorization: Bearer $USER1_TOKEN\" http://localhost:8080/"
echo ""

echo "Standard User 2:"
USER2_TOKEN=$(./jwt-signer -secret="$JWT_SECRET" -preset=user2)
echo "  AccountID: 67890, Role: user"  
echo "  Token: $USER2_TOKEN"
echo "  Test: curl -H \"Authorization: Bearer $USER2_TOKEN\" http://localhost:8080/"
echo ""

echo "Admin User:"
ADMIN_TOKEN=$(./jwt-signer -secret="$JWT_SECRET" -preset=admin1)
echo "  AccountID: 99999, Role: admin"
echo "  Token: $ADMIN_TOKEN"
echo "  Test: curl -H \"Authorization: Bearer $ADMIN_TOKEN\" http://localhost:8080/admin/status"
echo ""

echo "3. Generate custom tokens:"
echo ""

echo "Custom User (short duration):"
CUSTOM_TOKEN=$(./jwt-signer -secret="$JWT_SECRET" -user="testuser" -account=55555 -role="user" -duration="1h")
echo "  AccountID: 55555, Role: user, Duration: 1h"
echo "  Token: $CUSTOM_TOKEN"
echo ""

echo "Custom Admin (long duration):"
CUSTOM_ADMIN=$(./jwt-signer -secret="$JWT_SECRET" -user="superadmin" -account=11111 -role="admin" -duration="7d")
echo "  AccountID: 11111, Role: admin, Duration: 7d"
echo "  Token: $CUSTOM_ADMIN"
echo ""

echo "4. Different output formats:"
echo ""

echo "Header format:"
./jwt-signer -secret="$JWT_SECRET" -preset=user1 -output=header
echo ""

echo "Curl format:"
./jwt-signer -secret="$JWT_SECRET" -preset=user1 -output=curl
echo ""

echo "JSON format:"
./jwt-signer -secret="$JWT_SECRET" -preset=user1 -output=json