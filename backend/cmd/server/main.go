package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/user/transaction-system/internal/database"
	"github.com/user/transaction-system/internal/handler"
	"github.com/user/transaction-system/internal/kafka"
	"github.com/user/transaction-system/internal/repository"
)

func main() {
	godotenv.Load(".env")
	godotenv.Load("../.env")
	godotenv.Load("../../.env")
	godotenv.Load("../../../.env")

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = "user:password@tcp(localhost:3306)/transactions_db?parseTime=true"
	}

	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}

	// 1. Database & Repository
	db, err := database.NewMySQLConnection(dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := repository.NewTransactionRepository(db)

	// 2. Handlers
	restHandler := handler.NewRestHandler(repo)
	wsHandler := handler.NewWebSocketHandler()
	go wsHandler.BroadcastMessages()

	// 3. Kafka Consumer for Real-time events
	consumer := kafka.NewConsumer([]string{kafkaBroker}, "transaction-events", "websocket-group")
	msgChan := make(chan []byte)

	go consumer.ReadMessages(context.Background(), msgChan)

	go func() {
		for msg := range msgChan {
			log.Printf("Received Kafka message: %s", string(msg))
			wsHandler.SendMessage(msg)
		}
	}()

	// 4. Gin Router
	r := gin.Default()

	// CORS middleware for frontend (adjust for production)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/ws", wsHandler.HandleConnection)
	
	api := r.Group("/api")
	{
		api.POST("/transactions", restHandler.CreateTransaction)
		api.GET("/transactions", restHandler.GetTransactions)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
