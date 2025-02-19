package grpc

import (
	"context"
	"fmt"
	"kafka-gateway/internal/kafka"
	pb "kafka-gateway/proto/gen"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.UnimplementedKafkaGatewayServiceServer
	kafkaClient *kafka.Client
	grpcServer  *grpc.Server
}

func NewServer(kafkaClient *kafka.Client) *Server {
	grpcServer := grpc.NewServer()
	server := &Server{
		kafkaClient: kafkaClient,
		grpcServer:  grpcServer,
	}
	pb.RegisterKafkaGatewayServiceServer(grpcServer, server)
	reflection.Register(grpcServer)
	return server
}

func (s *Server) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

// Service implementations
func (s *Server) HealthCheck(ctx context.Context, _ *emptypb.Empty) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status: "healthy",
	}, nil
}

func (s *Server) PublishMessage(ctx context.Context, req *pb.PublishMessageRequest) (*pb.PublishMessageResponse, error) {
	var key []byte
	if req.Message.Key != "" {
		key = []byte(req.Message.Key)
	}

	err := s.kafkaClient.PublishMessage(req.Topic, key, []byte(req.Message.Value))
	if err != nil {
		return nil, err
	}

	return &pb.PublishMessageResponse{
		Status:  "success",
		Message: "Message published successfully",
		Topic:   req.Topic,
	}, nil
}

func (s *Server) ListTopics(ctx context.Context, _ *emptypb.Empty) (*pb.ListTopicsResponse, error) {
	topics, err := s.kafkaClient.ListTopics()
	if err != nil {
		return nil, err
	}

	return &pb.ListTopicsResponse{
		Topics: topics,
	}, nil
}

func (s *Server) GetTopicPartitions(ctx context.Context, req *pb.GetTopicPartitionsRequest) (*pb.GetTopicPartitionsResponse, error) {
	partitions, err := s.kafkaClient.GetTopicPartitions(req.Topic)
	if err != nil {
		return nil, err
	}

	return &pb.GetTopicPartitionsResponse{
		Topic:      req.Topic,
		Partitions: partitions,
	}, nil
}

func (s *Server) CreateTopic(ctx context.Context, req *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error) {
	err := s.kafkaClient.CreateTopic(
		req.Topic,
		req.Config.NumPartitions,
		int16(req.Config.ReplicationFactor),
	)
	if err != nil {
		return nil, err
	}

	return &pb.CreateTopicResponse{
		Status:  "success",
		Message: "Topic created successfully",
		Topic:   req.Topic,
	}, nil
}
