package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/user/transaction-system/internal/database"
	"github.com/user/transaction-system/internal/kafka"
	"github.com/user/transaction-system/internal/repository"
)

// Gunakan struct yang sama dengan repository agar data tidak hilang (seperti created_at)
type Transaction struct {
	ID          string  `json:"id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

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
		var tx Transaction
		if err := json.Unmarshal(msg, &tx); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			continue
		}

		// Jangan proses kalau statusnya sudah SUCCESS (biar tidak looping)
		if tx.Status == "SUCCESS" {
			continue
		}

		log.Printf("Processing transaction: %s", tx.ID)

		// Update status ke SUCCESS di Database
		if err := repo.UpdateTransactionStatus(context.Background(), tx.ID, "SUCCESS"); err != nil {
			log.Printf("Failed to update database for %s: %v", tx.ID, err)
			continue
		}

		// Kirim balik ke Kafka dengan data yang lengkap
		tx.Status = "SUCCESS"
		updatedMsg, _ := json.Marshal(tx)
		if err := producer.PublishMessage(context.Background(), tx.ID, string(updatedMsg)); err != nil {
			log.Printf("Failed to publish success event for %s: %v", tx.ID, err)
		}

		log.Printf("Transaction %s is now SUCCESS", tx.ID)
	}
}
