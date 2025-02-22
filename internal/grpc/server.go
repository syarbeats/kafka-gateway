package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"kafka-gateway/internal/config"
	"kafka-gateway/internal/kafka"
	pb "kafka-gateway/proto/gen"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.UnimplementedKafkaGatewayServiceServer
	kafkaClient *kafka.Client
	grpcServer  *grpc.Server
	config      *config.Config
}

func NewServer(kafkaClient *kafka.Client, cfg *config.Config) *Server {
	var opts []grpc.ServerOption

	if cfg.Server.TLS.Enabled {
		// Load CA certificate
		caCert, err := ioutil.ReadFile(cfg.Server.TLS.CACert)
		if err != nil {
			panic(fmt.Sprintf("failed to read CA certificate: %v", err))
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			panic("failed to append CA certificate")
		}

		// Load server certificate and private key
		cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.ServerCert, cfg.Server.TLS.ServerKey)
		if err != nil {
			panic(fmt.Sprintf("failed to load server certificate and key: %v", err))
		}

		// Create TLS credentials
		tlsConfig := &tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{cert},
			ClientCAs:    certPool,
		}

		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	grpcServer := grpc.NewServer(opts...)
	server := &Server{
		kafkaClient: kafkaClient,
		grpcServer:  grpcServer,
		config:      cfg,
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
