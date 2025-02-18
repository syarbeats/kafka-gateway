package kafka

import (
	"fmt"
	"kafka-gateway/internal/config"
	"sync"

	"github.com/Shopify/sarama"
)

type Client struct {
	config   *config.KafkaConfig
	producer sarama.SyncProducer
	admin    sarama.ClusterAdmin
	mu       sync.RWMutex
}

func NewClient(cfg config.KafkaConfig) (*Client, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	// Configure SASL if enabled
	if cfg.SecurityProtocol == "SASL_SSL" || cfg.SecurityProtocol == "SASL_PLAINTEXT" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = cfg.SaslUsername
		config.Net.SASL.Password = cfg.SaslPassword
		config.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SaslMechanism)

		if cfg.SecurityProtocol == "SASL_SSL" {
			config.Net.TLS.Enable = true
		}
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