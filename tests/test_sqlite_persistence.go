package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// MessageRequest represents the request body for publishing a message
type MessageRequest struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value"`
}

// Response represents a generic API response
type Response struct {
	Status   string    `json:"status,omitempty"`
	Message  string    `json:"message,omitempty"`
	Topic    string    `json:"topic,omitempty"`
	Error    string    `json:"error,omitempty"`
	Count    int       `json:"count,omitempty"`
	Messages []Message `json:"messages,omitempty"`
}

// Message represents a stored Kafka message
type Message struct {
	ID        int64     `json:"id"`
	Topic     string    `json:"topic"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Partition int32     `json:"partition"`
	Offset    int64     `json:"offset"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	fmt.Println("Starting SQLite persistence test...")
	fmt.Println("Loading certificates...")

	// Load client certificate
	cert, err := tls.LoadX509KeyPair("../certs/client/client.crt", "../certs/client/client.key")
	if err != nil {
		fmt.Printf("Error loading client certificate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Client certificate loaded successfully")

	// Load CA certificate
	caCert, err := ioutil.ReadFile("../certs/ca/ca.crt")
	if err != nil {
		fmt.Printf("Error loading CA certificate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("CA certificate loaded successfully")

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		fmt.Println("Error appending CA certificate")
		os.Exit(1)
	}
	fmt.Println("CA certificate appended to pool")

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Skip hostname verification for testing
	}
	fmt.Println("TLS config created")

	// Create HTTP client with TLS
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	fmt.Println("HTTP client created with TLS config")

	// Test topic
	topic := "test-topic-" + fmt.Sprintf("%d", time.Now().Unix())
	fmt.Printf("Using test topic: %s\n", topic)

	// Test message
	testMessage := MessageRequest{
		Key:   "test-key",
		Value: fmt.Sprintf("Test message at %s", time.Now().Format(time.RFC3339)),
	}
	fmt.Printf("Test message created: Key=%s, Value=%s\n", testMessage.Key, testMessage.Value)

	// 1. Publish message
	fmt.Println("\n1. Publishing message to Kafka...")
	publishURL := fmt.Sprintf("https://localhost:8080/api/v1/publish/%s", topic)
	fmt.Printf("Publish URL: %s\n", publishURL)

	messageJSON, err := json.Marshal(testMessage)
	if err != nil {
		fmt.Printf("Error marshaling message: %v\n", err)
		os.Exit(1)
	}

	// Verify JSON format
	var verifyMsg MessageRequest
	if err := json.Unmarshal(messageJSON, &verifyMsg); err != nil {
		fmt.Printf("Error verifying JSON format: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Message JSON: %s\n", string(messageJSON))

	req, err := http.NewRequest("POST", publishURL, bytes.NewBuffer(messageJSON))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	fmt.Println("Request created with Content-Type: application/json")

	fmt.Println("Sending request...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error publishing message: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("Response received with status code: %d\n", resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Response body: %s\n", string(body))

	var publishResponse Response
	if err := json.Unmarshal(body, &publishResponse); err != nil {
		fmt.Printf("Error unmarshaling response: %v\n", err)
		fmt.Println("Response body:", string(body))
		os.Exit(1)
	}
	fmt.Println("Response unmarshaled successfully")

	if publishResponse.Status != "success" {
		fmt.Printf("Failed to publish message: %s\n", publishResponse.Error)
		fmt.Println("Response body:", string(body))
		os.Exit(1)
	}

	fmt.Printf("Message published successfully to topic %s\n", topic)
	fmt.Printf("Response: %+v\n", publishResponse)

	// Wait a moment for the message to be processed
	fmt.Println("Waiting for message to be processed...")
	time.Sleep(2 * time.Second)

	// 2. Retrieve stored messages
	fmt.Println("\n2. Retrieving stored messages from SQLite...")
	retrieveURL := fmt.Sprintf("https://localhost:8080/api/v1/messages/%s", topic)
	fmt.Printf("Retrieve URL: %s\n", retrieveURL)

	req, err = http.NewRequest("GET", retrieveURL, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Request created")

	fmt.Println("Sending request...")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Error retrieving messages: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("Response received with status code: %d\n", resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Response body: %s\n", string(body))

	var retrieveResponse Response
	if err := json.Unmarshal(body, &retrieveResponse); err != nil {
		fmt.Printf("Error unmarshaling response: %v\n", err)
		fmt.Println("Response body:", string(body))
		os.Exit(1)
	}
	fmt.Println("Response unmarshaled successfully")

	// Check if messages were retrieved
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Failed to retrieve messages: %s\n", retrieveResponse.Error)
		fmt.Println("Response body:", string(body))
		os.Exit(1)
	}

	fmt.Printf("Retrieved %d messages from topic %s\n", retrieveResponse.Count, topic)

	// 3. Verify message content
	if retrieveResponse.Count == 0 {
		fmt.Println("No messages found in SQLite. Persistence may not be working correctly.")
		os.Exit(1)
	}

	fmt.Println("\n3. Verifying message content...")
	for i, msg := range retrieveResponse.Messages {
		fmt.Printf("Message %d:\n", i+1)
		fmt.Printf("  ID: %d\n", msg.ID)
		fmt.Printf("  Topic: %s\n", msg.Topic)
		fmt.Printf("  Key: %s\n", msg.Key)
		fmt.Printf("  Value: %s\n", msg.Value)
		fmt.Printf("  Partition: %d\n", msg.Partition)
		fmt.Printf("  Offset: %d\n", msg.Offset)
		fmt.Printf("  Timestamp: %s\n", msg.Timestamp)

		// Verify the message content matches what we sent
		if msg.Key == testMessage.Key && msg.Value == testMessage.Value {
			fmt.Println("✓ Message content verified successfully!")
		} else {
			fmt.Println("✗ Message content does not match what was sent!")
			fmt.Printf("  Expected Key: %s, Got: %s\n", testMessage.Key, msg.Key)
			fmt.Printf("  Expected Value: %s, Got: %s\n", testMessage.Value, msg.Value)
		}
	}

	fmt.Println("\nTest completed successfully! SQLite persistence is working correctly.")
}
