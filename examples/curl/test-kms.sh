#!/bin/bash
# Example: Testing KMS with IAM integration

set -e

PRINCIPAL="user:alice@example.com"
KMS_URL="http://localhost:8082"
PROJECT="test-project"
LOCATION="global"
KEYRING="test-keyring"
KEY="test-key"

echo "=== Testing KMS with principal: $PRINCIPAL ==="
echo

# Create a keyring
echo "1. Creating keyring '$KEYRING'..."
curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/$LOCATION/keyRings?keyRingId=$KEYRING" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" | jq .
echo

# Create a crypto key
echo "2. Creating crypto key '$KEY'..."
curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/$LOCATION/keyRings/$KEYRING/cryptoKeys?cryptoKeyId=$KEY" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d '{
    "purpose": "ENCRYPT_DECRYPT"
  }' | jq .
echo

# Encrypt data
echo "3. Encrypting data..."
PLAINTEXT=$(echo -n "my-secret-data" | base64)
CIPHERTEXT=$(curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/$LOCATION/keyRings/$KEYRING/cryptoKeys/$KEY:encrypt" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d "{
    \"plaintext\": \"$PLAINTEXT\"
  }" | jq -r '.ciphertext')
echo "Ciphertext: $CIPHERTEXT"
echo

# Decrypt data
echo "4. Decrypting data..."
RESPONSE=$(curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/$LOCATION/keyRings/$KEYRING/cryptoKeys/$KEY:decrypt" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d "{
    \"ciphertext\": \"$CIPHERTEXT\"
  }")
echo "$RESPONSE" | jq .

# Decode and display the plaintext
DECRYPTED=$(echo "$RESPONSE" | jq -r '.plaintext' | base64 -d)
echo "Decrypted plaintext: $DECRYPTED"
echo

# List keys
echo "5. Listing crypto keys..."
curl -s "$KMS_URL/v1/projects/$PROJECT/locations/$LOCATION/keyRings/$KEYRING/cryptoKeys" \
  -H "X-Emulator-Principal: $PRINCIPAL" | jq .
echo

echo "=== Test complete ==="
