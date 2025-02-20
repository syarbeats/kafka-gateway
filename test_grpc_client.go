package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "kafka-gateway/proto/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	// Connect to gRPC server
	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewKafkaGatewayServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test CreateTopic
	fmt.Println("Testing CreateTopic...")
	createResp, err := client.CreateTopic(ctx, &pb.CreateTopicRequest{
		Topic: "test-topic",
		Config: &pb.TopicConfig{
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	})
	if err != nil {
		log.Printf("CreateTopic failed: %v", err)
	} else {
		fmt.Printf("CreateTopic response: %+v\n", createResp)
	}

	// Wait a bit for topic creation to complete
	time.Sleep(2 * time.Second)

	// Test ListTopics
	fmt.Println("\nTesting ListTopics...")
	listResp, err := client.ListTopics(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("ListTopics failed: %v", err)
	} else {
		fmt.Printf("Available topics: %v\n", listResp.Topics)
	}
}
