# Kafka Gateway

A dual REST/gRPC API gateway for Apache Kafka operations.

## Features

- REST and gRPC APIs for Apache Kafka operations
- Mutual TLS (mTLS) authentication for all endpoints
- Swagger/OpenAPI documentation (mTLS protected)
- Metrics endpoint with Prometheus integration
- Health check endpoint
- Authentication support
- Graceful shutdown

## API Endpoints

The service provides both REST and gRPC endpoints for the following operations:

- Health check
- Publish message to topic
- List topics
- Get topic partitions
- Create topic

### REST API

REST endpoints are available at `https://localhost:8080/api/v1/` (requires mTLS):

- `GET /health` - Health check
- `POST /api/v1/publish/{topic}` - Publish message to topic
- `GET /api/v1/topics` - List topics
- `GET /api/v1/topics/{topic}/partitions` - Get topic partitions
- `POST /api/v1/topics/{topic}` - Create topic

### gRPC API

gRPC server runs on port 9090 with mTLS authentication required.

Example using `grpcurl` with mTLS:

```bash
# Health check
grpcurl -cert certs/client/client.crt -key certs/client/client.key -cacert certs/ca/ca.crt localhost:9090 kafka.gateway.v1.KafkaGatewayService/HealthCheck

# List topics
grpcurl -cert certs/client/client.crt -key certs/client/client.key -cacert certs/ca/ca.crt localhost:9090 kafka.gateway.v1.KafkaGatewayService/ListTopics

# Get topic partitions
grpcurl -cert certs/client/client.crt -key certs/client/client.key -cacert certs/ca/ca.crt -d '{"topic": "my-topic"}' localhost:9090 kafka.gateway.v1.KafkaGatewayService/GetTopicPartitions

# Create topic
grpcurl -cert certs/client/client.crt -key certs/client/client.key -cacert certs/ca/ca.crt -d '{"topic": "my-topic", "config": {"numPartitions": 3, "replicationFactor": 1}}' localhost:9090 kafka.gateway.v1.KafkaGatewayService/CreateTopic

# Publish message
grpcurl -cert certs/client/client.crt -key certs/client/client.key -cacert certs/ca/ca.crt -d '{"topic": "my-topic", "message": {"key": "key1", "value": "Hello, Kafka!"}}' localhost:9090 kafka.gateway.v1.KafkaGatewayService/PublishMessage
```

## API Documentation

Swagger UI is available at `https://localhost:8080/swagger/index.html` (requires mTLS)

### Setting Up Browser Certificates

To access Swagger UI, you need to configure your browser with the client certificate:

1. Import the CA certificate:
   ```bash
   # For Chrome/Safari:
   # - Open Settings > Privacy and Security > Security > Manage Certificates
   # - Go to Authorities tab
   # - Click Import and select certs/ca/ca.crt
   # - Check "Trust this certificate for identifying websites"
   ```

2. Import the client certificate:
   ```bash
   # The client certificate is packaged in PKCS12 format for easy browser import
   # Password: changeit
   
   # For Chrome/Safari:
   # - Open Settings > Privacy and Security > Security > Manage Certificates
   # - Go to Your Certificates tab
   # - Click Import and select certs/client/client.p12
   # - Enter the password: changeit
   ```

3. Access Swagger UI:
   - Navigate to `https://localhost:8080/swagger/index.html`
   - When prompted, select the imported client certificate
   - You should now see the Swagger UI documentation

## Configuration

Configuration is loaded from `config/config.yaml`. Example configuration:

```yaml
server:
  address: ":8080"
  tls:
    enabled: true  # Must be true for secure operation
    ca_cert: "certs/ca/ca.crt"
    server_cert: "certs/server/server.crt"
    server_key: "certs/server/server.key"

kafka:
  brokers:
    - "localhost:9092"
  consumer_group: "kafka-gateway"
  security_protocol: "PLAINTEXT"
  sasl_mechanism: ""
  sasl_username: ""
  sasl_password: ""

auth:
  enabled: false
  secret: ""
```

## Development

### Prerequisites

- Go 1.22 or later
- Protocol Buffers compiler (protoc)
- Kafka cluster
- OpenSSL (for generating certificates)

### Certificate Generation

Before running the service, generate the required certificates:

```bash
# Make the script executable
chmod +x scripts/generate_certs.sh

# Generate certificates
./scripts/generate_certs.sh

# Create PKCS12 file for browser import
openssl pkcs12 -export -out certs/client/client.p12 \
  -inkey certs/client/client.key \
  -in certs/client/client.crt \
  -certfile certs/ca/ca.crt \
  -passout pass:changeit
```

This will create:
- CA certificate and private key
- Server certificate and private key
- Client certificate and private key
- Client certificate in PKCS12 format for browser import

### Building

```bash
# Install tools
make install-tools

# Generate gRPC code
make proto

# Build the service
go build -o kafka-gateway cmd/gateway/main.go
```

### Running

```bash
# Run the service (TLS is required)
./kafka-gateway

# For testing, you can use the provided test client
go run test_grpc_client.go
```

## Metrics

Prometheus metrics are available at `https://localhost:8080/metrics` (requires mTLS)