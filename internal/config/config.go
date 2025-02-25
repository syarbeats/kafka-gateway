package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Kafka   KafkaConfig   `mapstructure:"kafka"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Storage StorageConfig `mapstructure:"storage"`
}

type ServerConfig struct {
	Address string    `mapstructure:"address"`
	TLS     TLSConfig `mapstructure:"tls"`
}

type TLSConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	CACert     string `mapstructure:"ca_cert"`
	ServerCert string `mapstructure:"server_cert"`
	ServerKey  string `mapstructure:"server_key"`
}

type KafkaConfig struct {
	Brokers          []string        `mapstructure:"brokers"`
	ConsumerGroup    string          `mapstructure:"consumer_group"`
	SecurityProtocol string          `mapstructure:"security_protocol"`
	TLS              *KafkaTLSConfig `mapstructure:"tls"`
}

type KafkaTLSConfig struct {
	CACert     string `mapstructure:"ca_cert"`
	ClientCert string `mapstructure:"client_cert"`
	ClientKey  string `mapstructure:"client_key"`
}

type AuthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Secret  string `mapstructure:"secret"`
}

type StorageConfig struct {
	SQLite SQLiteConfig `mapstructure:"sqlite"`
}

type SQLiteConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	DBPath    string `mapstructure:"db_path"`
	TableName string `mapstructure:"table_name"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.tls.enabled", false)
	viper.SetDefault("server.tls.ca_cert", "certs/ca/ca.crt")
	viper.SetDefault("server.tls.server_cert", "certs/server/server.crt")
	viper.SetDefault("server.tls.server_key", "certs/server/server.key")
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.consumer_group", "kafka-gateway")
	viper.SetDefault("kafka.security_protocol", "PLAINTEXT")
	viper.SetDefault("auth.enabled", false)
	viper.SetDefault("storage.sqlite.enabled", true)
	viper.SetDefault("storage.sqlite.db_path", "./data/messages.db")
	viper.SetDefault("storage.sqlite.table_name", "kafka_messages")

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
