// consumer/queue_consumer.go
package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"websocket-messaging/internal/rabbitmq"
)

type QueueConsumer interface {
	ProcessMessage(ctx context.Context, message string) (bool, error)
	ProcessBulkMessage(ctx context.Context, messages []string) (bool, error)
	GetConsumerName() string
}

type BufferedConsumer struct {
	rabbitMQ     *rabbitmq.RabbitMQ
	buffer       chan string
	bufferSize   int
	consumerImpl QueueConsumer
}

func NewBufferedConsumer(rabbitMQ *rabbitmq.RabbitMQ, consumer QueueConsumer, bufferSize int) *BufferedConsumer {
	return &BufferedConsumer{
		rabbitMQ:     rabbitMQ,
		buffer:       make(chan string, bufferSize),
		bufferSize:   bufferSize,
		consumerImpl: consumer,
	}
}

func (bc *BufferedConsumer) Start(queueName string) {
	go bc.consumeMessages()

	for {
		// Start consuming messages from RabbitMQ
		err := bc.rabbitMQ.Consume(queueName, func(msg string) {
			// Handler function for each received message
			// Push received messages to buffer for later processing
			bc.buffer <- msg
		})
		if err != nil {
			// log.Println("Error receiving messages:", err)
		}
	}
}

func (bc *BufferedConsumer) consumeMessages() {
	for {
		wg := &sync.WaitGroup{}
		for i := 0; i < bc.bufferSize; i++ {
			msg := <-bc.buffer
			wg.Add(1)
			go bc.processSingleMessage(msg, wg)
		}
		wg.Wait()
	}
}

func (bc *BufferedConsumer) processSingleMessage(msg string, wg *sync.WaitGroup) {
	defer wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println("Procesing message : :=")
	_, err := bc.consumerImpl.ProcessMessage(ctx, msg)
	if err != nil {
		log.Println("Failed to process message:", err)
	}
}
