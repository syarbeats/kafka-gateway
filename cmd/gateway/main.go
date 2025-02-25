package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "kafka-gateway/docs"
	"kafka-gateway/internal/config"
	grpcserver "kafka-gateway/internal/grpc"
	"kafka-gateway/internal/handler"
	"kafka-gateway/internal/kafka"
	"kafka-gateway/internal/middleware"
	"kafka-gateway/internal/storage"
	pb "kafka-gateway/proto/gen"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// @title           Kafka Gateway API
// @version         1.0
// @description     A REST API gateway for Apache Kafka operations
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /
// @schemes   https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize SQLite store if enabled
	var sqliteStore *storage.SQLiteStore
	if cfg.Storage.SQLite.Enabled {
		logger.Info("Initializing SQLite storage")
		if store, err := storage.NewSQLiteStore(cfg.Storage.SQLite); err != nil {
			logger.Warn("Failed to initialize SQLite storage, message persistence will be disabled", zap.Error(err))
		} else {
			sqliteStore = store
			logger.Info("SQLite storage initialized successfully",
				zap.String("db_path", cfg.Storage.SQLite.DBPath),
				zap.String("table", cfg.Storage.SQLite.TableName))
		}
	} else {
		logger.Info("SQLite storage is disabled")
	}

	// Initialize Kafka client
	var kafkaClient *kafka.Client
	if client, err := kafka.NewClient(cfg.Kafka, sqliteStore); err != nil {
		logger.Warn("Failed to create Kafka client, API endpoints will not be functional", zap.Error(err))
	} else {
		kafkaClient = client
		defer kafkaClient.Close()
	}

	// Start gRPC server
	grpcAddr := ":9090" // gRPC server address
	grpcServer := grpcserver.NewServer(kafkaClient, cfg)
	go func() {
		logger.Info("Starting gRPC server", zap.String("address", grpcAddr))
		if err := grpcServer.Start(9090); err != nil {
			logger.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	// Initialize gRPC-Gateway mux
	ctx := context.Background()
	gwmux := runtime.NewServeMux()

	// Register gRPC-Gateway handlers with TLS
	var opts []grpc.DialOption
	if cfg.Server.TLS.Enabled {
		// Load CA certificate
		caCert, err := ioutil.ReadFile(cfg.Server.TLS.CACert)
		if err != nil {
			logger.Fatal("Failed to read CA certificate", zap.Error(err))
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			logger.Fatal("Failed to append CA certificate")
		}

		// Load client certificate and private key
		cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.ServerCert, cfg.Server.TLS.ServerKey)
		if err != nil {
			logger.Fatal("Failed to load client certificate", zap.Error(err))
		}

		// Create TLS credentials
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      certPool,
		}

		opts = []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	} else {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	if err := pb.RegisterKafkaGatewayServiceHandlerFromEndpoint(
		ctx, gwmux, grpcAddr, opts,
	); err != nil {
		logger.Fatal("Failed to register gateway handler", zap.Error(err))
	}

	// Initialize Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Metrics())
	router.Use(middleware.CORS())

	// Add authentication middleware if enabled
	if cfg.Auth.Enabled {
		router.Use(middleware.Auth(cfg.Auth))
	}

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint
	router.GET("/health", handler.HealthCheck)

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Legacy REST API endpoints (only if Kafka client is available)
	if kafkaClient != nil {
		api := router.Group("/api/v1")
		{
			api.POST("/publish/:topic", handler.PublishMessage(kafkaClient))
			api.GET("/topics", handler.ListTopics(kafkaClient))
			api.GET("/topics/:topic/partitions", handler.GetTopicPartitions(kafkaClient))
			api.POST("/topics/:topic", handler.CreateTopic(kafkaClient))

			// Add endpoint to retrieve stored messages if SQLite is enabled
			if kafkaClient.GetStore() != nil {
				api.GET("/messages/:topic", handler.GetStoredMessages(kafkaClient))
			}
		}
	}

	// gRPC-Gateway endpoints
	router.Any("/grpc/v1/*path", gin.WrapH(gwmux))

	// Create HTTPS server
	srv := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: router,
	}

	// Configure TLS
	if !cfg.Server.TLS.Enabled {
		logger.Fatal("TLS must be enabled for secure operation")
	}

	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.ServerCert, cfg.Server.TLS.ServerKey)
	if err != nil {
		logger.Fatal("Failed to load server certificate", zap.Error(err))
	}

	// Load CA cert pool
	caCert, err := ioutil.ReadFile(cfg.Server.TLS.CACert)
	if err != nil {
		logger.Fatal("Failed to read CA certificate", zap.Error(err))
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		logger.Fatal("Failed to append CA certificate")
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}
	srv.TLSConfig = tlsConfig

	// Start HTTPS server
	go func() {
		logger.Info("Starting HTTPS server", zap.String("address", srv.Addr))
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTPS server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	logger.Info("Shutting down servers...")

	// Stop gRPC server
	grpcServer.Stop()

	// Shutdown HTTPS server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Servers exited")
}
