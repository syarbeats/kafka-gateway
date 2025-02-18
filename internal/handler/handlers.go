package handler

import (
	"io"
	"kafka-gateway/internal/kafka"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MessageRequest struct {
	Key   string `json:"key,omitempty" example:"user-123"`
	Value string `json:"value" binding:"required" example:"Hello, Kafka!"`
}

// @Summary Health check endpoint
// @Description Get the health status of the service
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// @Summary Publish message to Kafka topic
// @Description Publish a message to a specified Kafka topic
// @Tags kafka
// @Accept json
// @Produce json
// @Param topic path string true "Topic name"
// @Param message body MessageRequest true "Message to publish"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/publish/{topic} [post]
func PublishMessage(client *kafka.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		topic := c.Param("topic")
		if topic == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "topic is required"})
			return
		}

		var msg MessageRequest
		if err := c.ShouldBindJSON(&msg); err != nil {
			if err == io.EOF {
				c.JSON(http.StatusBadRequest, gin.H{"error": "request body is required"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var key []byte
		if msg.Key != "" {
			key = []byte(msg.Key)
		}

		err := client.PublishMessage(topic, key, []byte(msg.Value))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Message published successfully",
			"topic":   topic,
		})
	}
}

// @Summary List all Kafka topics
// @Description Get a list of all available Kafka topics
// @Tags kafka
// @Produce json
// @Success 200 {object} map[string][]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/topics [get]
func ListTopics(client *kafka.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		topics, err := client.ListTopics()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"topics": topics,
		})
	}
}

// @Summary Get topic partitions
// @Description Get partition information for a specific Kafka topic
// @Tags kafka
// @Produce json
// @Param topic path string true "Topic name"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/topics/{topic}/partitions [get]
func GetTopicPartitions(client *kafka.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		topic := c.Param("topic")
		if topic == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "topic is required"})
			return
		}

		partitions, err := client.GetTopicPartitions(topic)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"topic":      topic,
			"partitions": partitions,
		})
	}
}
