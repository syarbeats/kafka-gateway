# Kafka Gateway

A high-performance Apache Kafka gateway service written in Go that provides REST APIs for interacting with Kafka clusters.

## Features

- RESTful API for publishing messages to Kafka topics
- Topic management endpoints
- Authentication support
- Prometheus metrics
- Structured logging with Zap
- Health check endpoint
- SASL/SSL support for secure connections

## Prerequisites

- Go 1.21 or higher
- Apache Kafka cluster
- Docker (optional, for containerized deployment)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/kafka-gateway.git
cd kafka-gateway
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o kafka-gateway ./cmd/gateway
```

## Configuration

Copy the example configuration file and modify it according to your needs:

```bash
cp config/config.yaml config/config.local.yaml
```

Configuration options:

```yaml
server:
  address: ":8080"  # Server listen address

kafka:
  brokers:          # List of Kafka brokers
    - "localhost:9092"
  consumer_group: "kafka-gateway"
  security_protocol: "PLAINTEXT"  # PLAINTEXT, SASL_PLAINTEXT, or SASL_SSL
  sasl_mechanism: ""             # PLAIN, SCRAM-SHA-256, or SCRAM-SHA-512
  sasl_username: ""
  sasl_password: ""

auth:
  enabled: false    # Enable/disable authentication
  secret: ""        # JWT secret key when auth is enabled
```

## API Endpoints

### Health Check
```
GET /health
Response: {"status": "healthy"}
```

### Metrics
```
GET /metrics
Response: Prometheus metrics
```

### Publish Message
```
POST /api/v1/publish/{topic}
Body: {
  "key": "optional-message-key",
  "value": "message-content"
}
```

### List Topics
```
GET /api/v1/topics
Response: {
  "topics": ["topic1", "topic2", ...]
}
```

### Get Topic Partitions
```
GET /api/v1/topics/{topic}/partitions
Response: {
  "topic": "topic-name",
  "partitions": [0, 1, 2, ...]
}
```

## Authentication

When authentication is enabled, include the JWT token in the Authorization header:
```
Authorization: Bearer your-secret-token
```

## Metrics

The gateway exposes Prometheus metrics at the `/metrics` endpoint, including:
- Total HTTP requests
- HTTP request duration
- Request status codes

## Example Usage

Publishing a message:
```bash
curl -X POST \
  http://localhost:8080/api/v1/publish/my-topic \
  -H 'Content-Type: application/json' \
  -d '{
    "key": "user-123",
    "value": "Hello, Kafka!"
  }'
```

Listing topics:
```bash
curl http://localhost:8080/api/v1/topics
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.