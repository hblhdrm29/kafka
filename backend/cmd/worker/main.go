package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/user/transaction-system/internal/database"
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

	db, err := database.NewMySQLConnection(dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := repository.NewTransactionRepository(db)
	producer := kafka.NewProducer([]string{kafkaBroker}, "transaction-events")
	defer producer.Close()

	// Connect to RabbitMQ for bidirectional sync
	rmqUser := os.Getenv("RABBITMQ_USER")
	rmqPass := os.Getenv("RABBITMQ_PASS")
	rmqHost := os.Getenv("RABBITMQ_HOST")
	rmqPort := os.Getenv("RABBITMQ_PORT")
	if rmqUser == "" {
		rmqUser = "guest"
	}
	if rmqPass == "" {
		rmqPass = "guest"
	}
	if rmqHost == "" {
		rmqHost = "127.0.0.1"
	}
	if rmqPort == "" {
		rmqPort = "5672"
	}

	rmqURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", rmqUser, rmqPass, rmqHost, rmqPort)
	var rmqConn *amqp.Connection
	for i := 0; i < 10; i++ {
		rmqConn, err = amqp.Dial(rmqURL)
		if err == nil {
			break
		}
		log.Println("Waiting for RabbitMQ...")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("Warning: Could not connect to RabbitMQ: %v (Kafka-only mode)", err)
	}

	var rmqCh *amqp.Channel
	if rmqConn != nil {
		defer rmqConn.Close()
		rmqCh, err = rmqConn.Channel()
		if err != nil {
			log.Printf("Warning: Could not open RabbitMQ channel: %v", err)
		} else {
			defer rmqCh.Close()
			rmqCh.ExchangeDeclare("transaction_exchange", "fanout", true, false, false, false, nil)
		}
	}

	log.Println("Outbox worker started (Kafka + RabbitMQ)...")

	ctx := context.Background()
	for {
		events, err := repo.FetchPendingOutboxEvents(ctx, 50)
		if err != nil {
			log.Printf("Error fetching outbox events: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, event := range events {
			// 1. Publish to Kafka
			err := producer.PublishMessage(ctx, event.AggregateID, event.Payload)
			if err != nil {
				log.Printf("Failed to publish to Kafka %s: %v", event.ID, err)
				continue
			}

			// 2. Publish to RabbitMQ (for Web 1 & Web 2)
			if rmqCh != nil {
				rmqCh.Publish("transaction_exchange", "", false, false, amqp.Publishing{
					ContentType: "application/json",
					Body:        []byte(event.Payload),
					Type:        "TRANSACTION_CREATED",
				})
				log.Printf("Published event %s to Kafka + RabbitMQ", event.ID)
			} else {
				log.Printf("Published event %s to Kafka only", event.ID)
			}

			// Mark as processed
			err = repo.MarkOutboxEventProcessed(ctx, event.ID)
			if err != nil {
				log.Printf("Failed to mark event %s as processed: %v", event.ID, err)
			}
		}

		time.Sleep(1 * time.Second)
	}
}
