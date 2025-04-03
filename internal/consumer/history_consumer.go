// consumer/message_consumer.go
package consumer

import (
	"context"
	"log"
)

type HistoryConsumer struct{}

// NewMessageConsumer creates a new instance of MessageConsumer
func GetHistoryConsumer() *MessageConsumer {
	return &MessageConsumer{}
}

func (hc *HistoryConsumer) ProcessMessage(ctx context.Context, message string) (bool, error) {
	log.Println("Processing message:", message)
	return true, nil
}

func (hc *HistoryConsumer) ProcessBulkMessage(ctx context.Context, messages []string) (bool, error) {
	log.Println("Processing bulk messages:", messages)
	return true, nil
}

func (hc *HistoryConsumer) GetConsumerName() string {
	return "HistoryConsumer"
}
