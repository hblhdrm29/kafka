package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer creates a Kafka consumer that only reads the newest messages (LastOffset).
func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		// StartOffset: kafka.LastOffset memastikan consumer memulai dari pesan paling baru di log.
		StartOffset: kafka.LastOffset,
	})

	return &Consumer{reader: r}
}

// ReadMessages continuously reads messages and pushes them to a channel
func (c *Consumer) ReadMessages(ctx context.Context, messageChan chan<- []byte) {
	defer c.reader.Close()
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled, exit gracefully
				return
			}
			log.Printf("Error reading message: %v", err)
			continue
		}
		messageChan <- m.Value
	}
}
