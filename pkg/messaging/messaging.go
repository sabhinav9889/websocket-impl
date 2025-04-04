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

func Init(redisHost, redisPort, username, password, rabbitMQURL string, enableWebSocket, enableConsumer, enableHistoryConsumer bool) *Messaging {
	redisClient := redis.NewRedis(redisHost, redisPort, username, password)
	rabbitMQ := rabbitmq.NewRabbitMQ(rabbitMQURL)

	var wsServer *websocket.WebSocketServer
	if enableWebSocket {
		wsServer = websocket.NewWebSocketServer(redisClient, rabbitMQ)
	}

	return &Messaging{
		wsServer:           wsServer,
		rabbitMQ:           rabbitMQ,
		runWebSocket:       enableWebSocket,
		runConsumer:        enableConsumer,
		runHistoryConsumer: enableHistoryConsumer,
	}
}

func (m *Messaging) StartWebSocketServer(port, queueName string) {
	if !m.runWebSocket {
		log.Println("WebSocket server is disabled")
		return
	}

	go m.wsServer.StartRedisMessageListener()

	log.Println("WebSocket Server running on port", port)
	m.wsServer.Start(port, queueName)

}

func (m *Messaging) StartConsumer(queueName string, bufferSize int, host, port, username, password string) {
	redisClient := redis.NewRedis(host, port, username, password)
	if !m.runConsumer {
		log.Println("Consumer is disabled")
		return
	}
	// Initialize the consumer
	messageConsumer := consumer.GetMessageConsumer(*redisClient)

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
