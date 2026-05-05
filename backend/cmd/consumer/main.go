package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
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

	// 1. Database & Repository
	db, err := database.NewMySQLConnection(dbDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := repository.NewTransactionRepository(db)

	// 2. Kafka Producer
	producer := kafka.NewProducer([]string{kafkaBroker}, "transaction-events")
	defer producer.Close()

	// 3. Kafka Consumer
	consumer := kafka.NewConsumer([]string{kafkaBroker}, "transaction-events", "business-logic-group")
	msgChan := make(chan []byte)

	log.Printf("Business Consumer started, listening for events...")
	go consumer.ReadMessages(context.Background(), msgChan)

	for msg := range msgChan {
		// Detect event type from payload
		var raw map[string]interface{}
		json.Unmarshal(msg, &raw)
		
		eventType, _ := raw["type"].(string)
		
		log.Printf("Received event from Kafka: %s", eventType)

		if eventType == "PRODUCT_CREATED" {
			var p repository.Product
			json.Unmarshal(msg, &p)
			log.Printf("Syncing product to Kafka DB: %s", p.Name)
			if err := repo.SyncProduct(context.Background(), p); err != nil {
				log.Printf("Failed to sync product: %v", err)
			}
			continue
		}

		// Default to transaction
		var tx repository.Transaction
		if err := json.Unmarshal(msg, &tx); err != nil {
			log.Printf("Error unmarshaling transaction: %v", err)
			continue
		}

		log.Printf("Syncing transaction to Kafka DB: %s", tx.ID)
		if err := repo.SyncTransaction(context.Background(), tx); err != nil {
			log.Printf("Failed to sync transaction: %v", err)
		}

		// Jika statusnya PENDING, proses jadi SUCCESS
		if tx.Status == "PENDING" {
			log.Printf("Processing transaction: %s", tx.ID)
			time.Sleep(2 * time.Second) // Simulasi proses
			
			if err := repo.UpdateTransactionStatus(context.Background(), tx.ID, "SUCCESS"); err != nil {
				log.Printf("Failed to update status for %s: %v", tx.ID, err)
				continue
			}

			// Kirim balik ke Kafka agar UI terupdate
			tx.Status = "SUCCESS"
			updatedMsg, _ := json.Marshal(tx)
			producer.PublishMessage(context.Background(), tx.ID, string(updatedMsg))
		}
	}
}
