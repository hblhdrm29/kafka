package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/user/transaction-system/internal/database"
	"github.com/user/transaction-system/internal/kafka"
	"github.com/user/transaction-system/internal/repository"
)

func main() {
	godotenv.Load(".env", "../.env")

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

	log.Println("Outbox worker started...")

	ctx := context.Background()
	for {
		// Poll for pending events
		events, err := repo.FetchPendingOutboxEvents(ctx, 50)
		if err != nil {
			log.Printf("Error fetching outbox events: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, event := range events {
			// Publish to Kafka
			err := producer.PublishMessage(ctx, event.AggregateID, event.Payload)
			if err != nil {
				log.Printf("Failed to publish event %s: %v", event.ID, err)
				continue
			}

			// Mark as processed
			err = repo.MarkOutboxEventProcessed(ctx, event.ID)
			if err != nil {
				log.Printf("Failed to mark event %s as processed: %v", event.ID, err)
				// It will be picked up again if we don't mark it, leading to at-least-once delivery (which is fine, Kafka consumer should handle idempotency if needed).
			} else {
				log.Printf("Successfully processed event %s for transaction %s", event.ID, event.AggregateID)
			}
		}

		time.Sleep(1 * time.Second)
	}
}
