package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	pb "kafka-gateway/proto/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	// Load client certificate and private key
	cert, err := tls.LoadX509KeyPair("certs/client/client.crt", "certs/client/client.key")
	if err != nil {
		log.Fatalf("Failed to load client certificate: %v", err)
	}

	// Load CA certificate
	caCert, err := ioutil.ReadFile("certs/ca/ca.crt")
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		log.Fatal("Failed to append CA certificate")
	}

	// Create TLS credentials
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		ServerName:   "localhost", // Must match the server's certificate CN
	}

	// Create connection with TLS credentials
	conn, err := grpc.Dial(
		"localhost:9090", // Changed to connect to gRPC port
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewKafkaGatewayServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test health check
	resp, err := client.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Health check response: %s\n", resp.Status)

	// Test list topics
	topics, err := client.ListTopics(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("List topics failed: %v", err)
	}
	fmt.Printf("Topics: %v\n", topics.Topics)

	// Test publish message
	pubResp, err := client.PublishMessage(ctx, &pb.PublishMessageRequest{
		Topic: "test-topic",
		Message: &pb.Message{
			Key:   "test-key",
			Value: "test-value",
		},
	})
	if err != nil {
		log.Fatalf("Publish message failed: %v", err)
	}
	fmt.Printf("Publish response: %s\n", pubResp.Status)
}
