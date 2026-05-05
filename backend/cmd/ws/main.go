package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/user/transaction-system/internal/handler"
	"github.com/user/transaction-system/internal/kafka"
)

func main() {
	godotenv.Load(".env")
	godotenv.Load("../.env")
	godotenv.Load("../../.env")
	godotenv.Load("../../../.env")

	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}

	// 1. Handlers
	wsHandler := handler.NewWebSocketHandler()
	go wsHandler.BroadcastMessages()

	// 2. Kafka Consumer for Real-time events
	consumer := kafka.NewConsumer([]string{kafkaBroker}, "transaction-events", "websocket-gateway-group")
	msgChan := make(chan []byte)

	go consumer.ReadMessages(context.Background(), msgChan)

	go func() {
		for msg := range msgChan {
			log.Printf("Received Kafka message: %s", string(msg))
			wsHandler.SendMessage(msg)
		}
	}()

	// 3. Gin Router for WebSocket only
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/ws", wsHandler.HandleConnection)

	port := os.Getenv("WS_PORT")
	if port == "" {
		port = "8086"
	}

	log.Printf("WebSocket Gateway starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run WebSocket server: %v", err)
	}
}
