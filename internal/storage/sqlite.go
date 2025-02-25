package storage

import (
	"database/sql"
	"fmt"
	"kafka-gateway/internal/config"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// MessageRecord represents a Kafka message stored in SQLite
type MessageRecord struct {
	ID        int64
	Topic     string
	Key       []byte
	Value     []byte
	Partition int32
	Offset    int64
	Timestamp time.Time
}

// SQLiteStore handles SQLite database operations
type SQLiteStore struct {
	db        *sql.DB
	tableName string
}

// NewSQLiteStore creates a new SQLite storage instance
func NewSQLiteStore(cfg config.SQLiteConfig) (*SQLiteStore, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("SQLite storage is disabled")
	}

	// Ensure directory exists
	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Set connection parameters
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	store := &SQLiteStore{
		db:        db,
		tableName: cfg.TableName,
	}

	// Initialize database schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// initSchema creates the necessary tables if they don't exist
func (s *SQLiteStore) initSchema() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic TEXT NOT NULL,
			message_key BLOB,
			message_value BLOB NOT NULL,
			partition INTEGER NOT NULL,
			offset INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_%s_topic ON %s (topic);
		CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s (timestamp);
	`, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName)

	_, err := s.db.Exec(query)
	return err
}

// SaveMessage stores a Kafka message in the database
func (s *SQLiteStore) SaveMessage(topic string, key, value []byte, partition int32, offset int64) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (topic, message_key, message_value, partition, offset, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, s.tableName)

	_, err := s.db.Exec(query, topic, key, value, partition, offset, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save message to SQLite: %w", err)
	}

	return nil
}

// GetMessagesByTopic retrieves messages for a specific topic
func (s *SQLiteStore) GetMessagesByTopic(topic string, limit, offset int) ([]MessageRecord, error) {
	query := fmt.Sprintf(`
		SELECT id, topic, message_key, message_value, partition, offset, timestamp
		FROM %s
		WHERE topic = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, s.tableName)

	rows, err := s.db.Query(query, topic, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var msg MessageRecord
		if err := rows.Scan(&msg.ID, &msg.Topic, &msg.Key, &msg.Value, &msg.Partition, &msg.Offset, &msg.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

// GetMessageCount returns the total number of stored messages
func (s *SQLiteStore) GetMessageCount() (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.tableName)

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}

	return count, nil
}
