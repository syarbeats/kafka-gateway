#!/bin/bash

# Create directories for certificates
mkdir -p certs/{ca,server,client}

# Generate CA private key and certificate
openssl genpkey -algorithm RSA -out certs/ca/ca.key
openssl req -new -x509 -key certs/ca/ca.key -out certs/ca/ca.crt -days 365 -subj "/CN=Kafka Gateway CA"

# Generate server private key and CSR
openssl genpkey -algorithm RSA -out certs/server/server.key
openssl req -new -key certs/server/server.key -out certs/server/server.csr -subj "/CN=localhost"

# Create server certificate config
cat > certs/server/server.conf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = localhost

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
EOF

# Sign server certificate with CA
openssl x509 -req \
    -in certs/server/server.csr \
    -CA certs/ca/ca.crt \
    -CAkey certs/ca/ca.key \
    -CAcreateserial \
    -out certs/server/server.crt \
    -days 365 \
    -extfile certs/server/server.conf \
    -extensions v3_req

# Generate client private key and CSR
openssl genpkey -algorithm RSA -out certs/client/client.key
openssl req -new -key certs/client/client.key -out certs/client/client.csr -subj "/CN=kafka-gateway-client"

# Sign client certificate with CA
openssl x509 -req -in certs/client/client.csr -CA certs/ca/ca.crt -CAkey certs/ca/ca.key -CAcreateserial -out certs/client/client.crt -days 365

# Set permissions
chmod 600 certs/ca/ca.key certs/server/server.key certs/client/client.key

echo "Certificates generated successfully in the certs directory"