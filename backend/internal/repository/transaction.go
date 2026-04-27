package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type OutboxEvent struct {
	ID            string    `json:"id"`
	AggregateType string    `json:"aggregate_type"`
	AggregateID   string    `json:"aggregate_id"`
	EventType     string    `json:"event_type"`
	Payload       string    `json:"payload"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type TransactionRepository struct {
	DB *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{DB: db}
}

// CreateTransaction atomically saves the transaction and the outbox event
func (r *TransactionRepository) CreateTransaction(ctx context.Context, amount float64, description string) (*Transaction, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Create Transaction
	transaction := &Transaction{
		ID:          uuid.New().String(),
		Amount:      amount,
		Description: description,
		Status:      "PENDING",
		CreatedAt:   time.Now(),
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO transactions (id, amount, description, status, created_at) VALUES (?, ?, ?, ?, ?)",
		transaction.ID, transaction.Amount, transaction.Description, transaction.Status, transaction.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// 2. Create Outbox Event
	payload, _ := json.Marshal(transaction)
	outboxEvent := &OutboxEvent{
		ID:            uuid.New().String(),
		AggregateType: "Transaction",
		AggregateID:   transaction.ID,
		EventType:     "TransactionCreated",
		Payload:       string(payload),
		Status:        "PENDING",
		CreatedAt:     time.Now(),
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		outboxEvent.ID, outboxEvent.AggregateType, outboxEvent.AggregateID, outboxEvent.EventType, outboxEvent.Payload, outboxEvent.Status, outboxEvent.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return transaction, nil
}

// FetchPendingOutboxEvents gets pending events and locks them (simple approach)
func (r *TransactionRepository) FetchPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	// For production, use FOR UPDATE SKIP LOCKED or similar if running multiple workers.
	// For MySQL 8, SKIP LOCKED is supported.
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, aggregate_type, aggregate_id, event_type, payload, status, created_at FROM outbox_events WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT ? FOR UPDATE SKIP LOCKED",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *TransactionRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE outbox_events SET status = 'PROCESSED' WHERE id = ?", id)
	return err
}

func (r *TransactionRepository) UpdateTransactionStatus(ctx context.Context, id string, status string) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE transactions SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *TransactionRepository) GetTransactions(ctx context.Context) ([]Transaction, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id, amount, description, status, created_at FROM transactions ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Description, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}
