package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"kafka-gateway/internal/config"
	"kafka-gateway/internal/storage"
	"sync"
	"time"

	"github.com/Shopify/sarama"
)

func createTLSConfig(tlsConfig *config.KafkaTLSConfig) (*tls.Config, error) {
	if tlsConfig == nil {
		return nil, fmt.Errorf("TLS config is required")
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair("certs/client/client.crt", "certs/client/client.key")
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile("certs/ca/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

type Client struct {
	config   *config.KafkaConfig
	producer sarama.SyncProducer
	admin    sarama.ClusterAdmin
	mu       sync.RWMutex
	store    *storage.SQLiteStore
}

func NewClient(cfg config.KafkaConfig, store *storage.SQLiteStore) (*Client, error) {
	config := sarama.NewConfig()

	// Producer configs
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	// Connection configs
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second
	config.Metadata.Retry.Max = 3
	config.Metadata.Retry.Backoff = 1 * time.Second

	// Configure TLS if enabled
	if cfg.SecurityProtocol == "SSL" {
		tlsConfig, err := createTLSConfig(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	// Create producer
	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	// Create admin client
	admin, err := sarama.NewClusterAdmin(cfg.Brokers, config)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}

	return &Client{
		config:   &cfg,
		producer: producer,
		admin:    admin,
		store:    store,
	}, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.producer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}
	if err := c.admin.Close(); err != nil {
		return fmt.Errorf("failed to close admin client: %w", err)
	}

	// Close SQLite store if it exists
	if c.store != nil {
		if err := c.store.Close(); err != nil {
			return fmt.Errorf("failed to close SQLite store: %w", err)
		}
	}

	return nil
}

func (c *Client) PublishMessage(topic string, key []byte, value []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(value),
	}
	if key != nil {
		msg.Key = sarama.ByteEncoder(key)
	}

	partition, offset, err := c.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Save message to SQLite if storage is enabled
	if c.store != nil {
		if err := c.store.SaveMessage(topic, key, value, partition, offset); err != nil {
			// Log the error but don't fail the publish operation
			fmt.Printf("Warning: Failed to save message to SQLite: %v\n", err)
		}
	}

	fmt.Printf("Message published to topic %s, partition %d, offset %d\n", topic, partition, offset)
	return nil
}

func (c *Client) ListTopics() ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	topics, err := c.admin.ListTopics()
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}

	topicList := make([]string, 0, len(topics))
	for topic := range topics {
		topicList = append(topicList, topic)
	}
	return topicList, nil
}

func (c *Client) GetTopicPartitions(topic string) ([]int32, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metadata, err := c.admin.DescribeTopics([]string{topic})
	if err != nil {
		return nil, fmt.Errorf("failed to get topic metadata: %w", err)
	}

	if len(metadata) == 0 {
		return nil, fmt.Errorf("topic not found: %s", topic)
	}

	partitions := make([]int32, len(metadata[0].Partitions))
	for i, partition := range metadata[0].Partitions {
		partitions[i] = partition.ID
	}
	return partitions, nil
}

func (c *Client) CreateTopic(topic string, numPartitions int32, replicationFactor int16) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	}

	err := c.admin.CreateTopic(topic, topicDetail, false)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	return nil
}

// GetStore returns the SQLite store instance
func (c *Client) GetStore() *storage.SQLiteStore {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.store
}
