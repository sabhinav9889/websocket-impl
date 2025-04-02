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
	wsServer     *websocket.WebSocketServer
	rabbitMQ     *rabbitmq.RabbitMQ
	runWebSocket bool
	runConsumer  bool
}

func Init(redisHost, redisPort, rabbitMQURL string, enableWebSocket, enableConsumer bool) *Messaging {
	redisClient := redis.NewRedis(redisHost, redisPort)
	rabbitMQ := rabbitmq.NewRabbitMQ(rabbitMQURL)

	var wsServer *websocket.WebSocketServer
	if enableWebSocket {
		wsServer = websocket.NewWebSocketServer(redisClient, rabbitMQ)
	}

	return &Messaging{
		wsServer:     wsServer,
		rabbitMQ:     rabbitMQ,
		runWebSocket: enableWebSocket,
		runConsumer:  enableConsumer,
	}
}

func (m *Messaging) StartWebSocketServer(port , queueName string) {
	if !m.runWebSocket {
		log.Println("WebSocket server is disabled")
		return
	}
	log.Println("WebSocket Server running on port", port)
	m.wsServer.Start(port, queueName)
}

func (m *Messaging) StartConsumer(queueName string, redisHost, redisPort string) {
	redisClient := redis.NewRedis(redisHost, redisPort)
	if !m.runConsumer {
		log.Println("Consumer is disabled")
		return
	}
	// Initialize the consumer
	messageConsumer := consumer.NewMessageConsumer(*redisClient)
	bufferSize := 100 // Define your buffer size

	// Initialize the BufferedConsumer
	bufferedConsumer := consumer.NewBufferedConsumer(m.rabbitMQ, messageConsumer, bufferSize)

	// Start the BufferedConsumer to consume messages from the queue
	log.Println("Starting consumer for queue:", queueName)
	bufferedConsumer.Start(queueName)

}
