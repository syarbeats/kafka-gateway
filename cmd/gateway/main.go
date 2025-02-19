package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "kafka-gateway/docs"
	"kafka-gateway/internal/config"
	"kafka-gateway/internal/handler"
	"kafka-gateway/internal/kafka"
	"kafka-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
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

	// Initialize Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Metrics())

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

	// Initialize Kafka client (optional for testing Swagger)
	var kafkaClient *kafka.Client
	if client, err := kafka.NewClient(cfg.Kafka); err != nil {
		logger.Warn("Failed to create Kafka client, API endpoints will not be functional", zap.Error(err))
	} else {
		kafkaClient = client
		defer kafkaClient.Close()
	}

	// API endpoints (only if Kafka client is available)
	if kafkaClient != nil {
		api := router.Group("/api/v1")
		{
			api.POST("/publish/:topic", handler.PublishMessage(kafkaClient))
			api.GET("/topics", handler.ListTopics(kafkaClient))
			api.GET("/topics/:topic/partitions", handler.GetTopicPartitions(kafkaClient))
			api.POST("/topics/:topic", handler.CreateTopic(kafkaClient))
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", zap.String("address", cfg.Server.Address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
