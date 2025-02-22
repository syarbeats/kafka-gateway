#!/bin/bash

echo "Testing mTLS configuration..."

# Test 1: Verify certificate files exist
echo -e "\n1. Checking certificate files..."
files=(
    "certs/ca/ca.crt"
    "certs/server/server.crt"
    "certs/server/server.key"
    "certs/client/client.crt"
    "certs/client/client.key"
    "certs/client/client.p12"
)

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ Found $file"
    else
        echo "✗ Missing $file"
    fi
done

# Test 2: Verify certificate validity
echo -e "\n2. Verifying certificates..."
echo "CA certificate info:"
openssl x509 -in certs/ca/ca.crt -text -noout | grep "Subject:\|Issuer:\|Not After"

echo -e "\nServer certificate info:"
openssl x509 -in certs/server/server.crt -text -noout | grep "Subject:\|Issuer:\|Not After\|DNS:\|IP Address:"

echo -e "\nClient certificate info:"
openssl x509 -in certs/client/client.crt -text -noout | grep "Subject:\|Issuer:\|Not After"

# Test 3: Test HTTPS endpoints with curl
echo -e "\n3. Testing HTTPS endpoints with mTLS..."

# Test Swagger UI endpoint
echo -e "\nTesting Swagger UI endpoint:"
curl --cacert certs/ca/ca.crt \
     --cert certs/client/client.crt \
     --key certs/client/client.key \
     -v https://localhost:8080/swagger/index.html

# Test health endpoint
echo -e "\nTesting health endpoint:"
curl --cacert certs/ca/ca.crt \
     --cert certs/client/client.crt \
     --key certs/client/client.key \
     -v https://localhost:8080/health

# Test topics endpoint
echo -e "\nTesting topics endpoint:"
curl --cacert certs/ca/ca.crt \
     --cert certs/client/client.crt \
     --key certs/client/client.key \
     -v https://localhost:8080/api/v1/topics

echo -e "\nTests completed. Check the output above for any errors."
echo "If all tests pass, you should be able to access Swagger UI at https://localhost:8080/swagger/index.html"
echo "Remember to import the certificates into your browser:"
echo "1. Import certs/ca/ca.crt as a trusted root certificate"
echo "2. Import certs/client/client.p12 as a client certificate (password: changeit)"