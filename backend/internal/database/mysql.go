package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQLConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	txSchema := `
	CREATE TABLE IF NOT EXISTS transactions (
		id VARCHAR(36) PRIMARY KEY,
		amount DECIMAL(10,2) NOT NULL,
		description TEXT NOT NULL,
		status ENUM('PENDING', 'SUCCESS', 'FAILED') DEFAULT 'PENDING',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	outboxSchema := `
	CREATE TABLE IF NOT EXISTS outbox_events (
		id VARCHAR(36) PRIMARY KEY,
		aggregate_type VARCHAR(255) NOT NULL,
		aggregate_id VARCHAR(36) NOT NULL,
		event_type VARCHAR(255) NOT NULL,
		payload JSON NOT NULL,
		status ENUM('PENDING', 'PROCESSED') DEFAULT 'PENDING',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(txSchema); err != nil {
		return err
	}

	if _, err := db.Exec(outboxSchema); err != nil {
		return err
	}

	log.Println("Database migration completed successfully.")
	return nil
}
