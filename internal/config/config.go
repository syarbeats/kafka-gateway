package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Kafka  KafkaConfig  `mapstructure:"kafka"`
	Auth   AuthConfig   `mapstructure:"auth"`
}

type ServerConfig struct {
	Address string `mapstructure:"address"`
}

type KafkaConfig struct {
	Brokers          []string `mapstructure:"brokers"`
	ConsumerGroup    string   `mapstructure:"consumer_group"`
	SecurityProtocol string   `mapstructure:"security_protocol"`
	SaslMechanism    string   `mapstructure:"sasl_mechanism"`
	SaslUsername     string   `mapstructure:"sasl_username"`
	SaslPassword     string   `mapstructure:"sasl_password"`
}

type AuthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Secret  string `mapstructure:"secret"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.consumer_group", "kafka-gateway")
	viper.SetDefault("kafka.security_protocol", "PLAINTEXT")
	viper.SetDefault("auth.enabled", false)

	// Read configuration
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}