// messaging.go
package messaging

import (
	"log"
	"websocket-messaging/internal/consumer"
	"websocket-messaging/internal/rabbitmq"
	"websocket-messaging/internal/redis"
	"websocket-messaging/internal/websocket"
)

type Messaging struct {
	wsServer           *websocket.WebSocketServer
	rabbitMQ           *rabbitmq.RabbitMQ
	runWebSocket       bool
	runConsumer        bool
	runHistoryConsumer bool
}

func Init(serverID, redisHost, redisPort, username, password, rabbitMQURL string, enableWebSocket, enableConsumer, enableHistoryConsumer bool) *Messaging {
	redisClient := redis.NewRedis(redisHost, redisPort, username, password)
	rabbitMQ := rabbitmq.NewRabbitMQ(rabbitMQURL)

	var wsServer *websocket.WebSocketServer
	if enableWebSocket {
		wsServer = websocket.NewWebSocketServer(serverID, redisClient)
	}

	return &Messaging{
		wsServer:           wsServer,
		rabbitMQ:           rabbitMQ,
		runWebSocket:       enableWebSocket,
		runConsumer:        enableConsumer,
		runHistoryConsumer: enableHistoryConsumer,
	}
}

func (m *Messaging) StartWebSocketServer(port string) {
	if !m.runWebSocket {
		log.Println("WebSocket server is disabled")
		return
	}
	log.Println("WebSocket Server running on port", port)
	m.wsServer.Start(port)
}

func (m *Messaging) StartConsumer(queueName string, bufferSize int) {
	if !m.runConsumer {
		log.Println("Consumer is disabled")
		return
	}
	// Initialize the consumer
	messageConsumer := consumer.GetMessageConsumer()

	// Initialize the BufferedConsumer
	bufferedConsumer := consumer.NewBufferedConsumer(m.rabbitMQ, messageConsumer, bufferSize)

	// Start the BufferedConsumer to consume messages from the queue
	log.Println("Starting consumer for queue:", queueName)
	bufferedConsumer.Start(queueName)

}

func (m *Messaging) StartHistoryConsumer(queueName string, bufferSize int) {
	if !m.runHistoryConsumer {
		log.Println("HistoryConsumer is disabled")
		return
	}
	// Initialize the consumer
	historyConsumer := consumer.GetHistoryConsumer()

	// Initialize the BufferedConsumer
	bufferedHistoryConsumer := consumer.NewBufferedConsumer(m.rabbitMQ, historyConsumer, bufferSize)

	// Start the BufferedConsumer to consume messages from the queue
	log.Println("Starting history consumer for queue:", queueName)
	bufferedHistoryConsumer.Start(queueName)

}
