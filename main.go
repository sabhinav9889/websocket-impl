package main

import (
	"os"

	"websocket-messaging/pkg/messaging"
)

func main() {

	// Load environment variables or set default values
	serverID := getEnv("SERVER_ID", "server-1")
	redisHost := getEnv("REDIS_HOST", "singapore-keyvalue.render.com")
	redisPort := getEnv("REDIS_PORT", "6379")
	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://rabbitmq_dev:f55da0077317b31bc7af5fde7d46f256@https://rabbitmq-z4fk.onrender.com:5672/")
	//dbURL := getEnv("DB_URL", "")
	webSocketPort := getEnv("WS_PORT", "8080")
	queueName := getEnv("RABBITMQ_QUEUE", "messages")
	enableWebSocket := getEnv("ENABLE_WEBSOCKET", "true") == "true"
	enableConsumer := getEnv("ENABLE_CONSUMER", "true") == "true"
	enableHistoryConsumer := getEnv("ENABLE_HISTORY_CONSUMER", "true") == "true"

	// Initialize Messaging
	msgService := messaging.Init(serverID, redisHost, redisPort, "red-cukv0hin91rc73b06iu0", "nHT8A1LNngoS7F1y0VilI1Fe9VKqbgX0", rabbitMQURL, enableWebSocket, enableConsumer, enableHistoryConsumer)

	// Start WebSocket Server if enabled
	if enableWebSocket {
		go msgService.StartWebSocketServer(webSocketPort)
	}

	// Start Consumer if enabled
	if enableConsumer {
		bufferSize := 100
		go msgService.StartConsumer(queueName, bufferSize)
	}

	// Keep the server running
	select {}
}

// getEnv retrieves environment variables or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
