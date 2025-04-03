// consumer/message_consumer.go
package consumer

import (
	"context"
	"log"
)

type MessageConsumer struct{}

// NewMessageConsumer creates a new instance of MessageConsumer
func GetMessageConsumer() *MessageConsumer {
	return &MessageConsumer{}
}

func (mc *MessageConsumer) ProcessMessage(ctx context.Context, message string) (bool, error) {
	log.Println("Processing message:", message)
	return true, nil
}

func (mc *MessageConsumer) ProcessBulkMessage(ctx context.Context, messages []string) (bool, error) {
	log.Println("Processing bulk messages:", messages)
	return true, nil
}

func (mc *MessageConsumer) GetConsumerName() string {
	return "MessageConsumer"
}
