# Kafka Gateway

A dual REST/gRPC API gateway for Apache Kafka operations.

## Features

- REST and gRPC APIs for Kafka operations
- Swagger/OpenAPI documentation
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

REST endpoints are available at `http://localhost:8080/api/v1/`:

- `GET /health` - Health check
- `POST /api/v1/publish/{topic}` - Publish message to topic
- `GET /api/v1/topics` - List topics
- `GET /api/v1/topics/{topic}/partitions` - Get topic partitions
- `POST /api/v1/topics/{topic}` - Create topic

### gRPC API

gRPC server runs on port 9090. You can use any gRPC client to interact with the service.

Example using `grpcurl`:

```bash
# Health check
grpcurl -plaintext localhost:9090 kafka.gateway.v1.KafkaGatewayService/HealthCheck

# List topics
grpcurl -plaintext localhost:9090 kafka.gateway.v1.KafkaGatewayService/ListTopics

# Get topic partitions
grpcurl -plaintext -d '{"topic": "my-topic"}' localhost:9090 kafka.gateway.v1.KafkaGatewayService/GetTopicPartitions

# Create topic
grpcurl -plaintext -d '{"topic": "my-topic", "config": {"numPartitions": 3, "replicationFactor": 1}}' localhost:9090 kafka.gateway.v1.KafkaGatewayService/CreateTopic

# Publish message
grpcurl -plaintext -d '{"topic": "my-topic", "message": {"key": "key1", "value": "Hello, Kafka!"}}' localhost:9090 kafka.gateway.v1.KafkaGatewayService/PublishMessage
```

## API Documentation

Swagger UI is available at `http://localhost:8080/swagger/index.html`

## Configuration

Configuration is loaded from `config/config.yaml`. Example configuration:

```yaml
server:
  address: ":8080"

kafka:
  brokers:
    - "localhost:9092"
  clientID: "kafka-gateway"

auth:
  enabled: false
  apiKey: "your-api-key"
```

## Development

### Prerequisites

- Go 1.22 or later
- Protocol Buffers compiler (protoc)
- Kafka cluster

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
./kafka-gateway
```

## Metrics

Prometheus metrics are available at `http://localhost:8080/metrics`