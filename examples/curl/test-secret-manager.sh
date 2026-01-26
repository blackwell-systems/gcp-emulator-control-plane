#!/bin/bash
# Example: Testing Secret Manager with IAM integration

set -e

PRINCIPAL="user:alice@example.com"
SM_URL="http://localhost:8081"
PROJECT="test-project"

echo "=== Testing Secret Manager with principal: $PRINCIPAL ==="
echo

# Create a secret
echo "1. Creating secret 'db-password'..."
curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d '{
    "secretId": "db-password",
    "replication": {
      "automatic": {}
    }
  }' | jq .
echo

# Add a secret version
echo "2. Adding secret version..."
SECRET_DATA=$(echo -n "my-secret-password" | base64)
curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets/db-password:addVersion" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d "{
    \"payload\": {
      \"data\": \"$SECRET_DATA\"
    }
  }" | jq .
echo

# Access the secret version
echo "3. Accessing secret version..."
RESPONSE=$(curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets/db-password/versions/1:access" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json")
echo "$RESPONSE" | jq .

# Decode and display the secret
SECRET=$(echo "$RESPONSE" | jq -r '.payload.data' | base64 -d)
echo "Decrypted secret: $SECRET"
echo

# List secrets
echo "4. Listing secrets..."
curl -s "$SM_URL/v1/projects/$PROJECT/secrets" \
  -H "X-Emulator-Principal: $PRINCIPAL" | jq .
echo

echo "=== Test complete ==="
