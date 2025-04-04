package main

import (
	"os"

	"websocket-messaging/pkg/messaging"
)

func main() {

	// Load environment variables or set default values
	// serverID := getEnv("SERVER_ID", "server-1")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisUserName := getEnv("REDIS_USERNAME", "")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	//dbURL := getEnv("DB_URL", "")
	webSocketPort := getEnv("WS_PORT", "8000")
	queueName := getEnv("RABBITMQ_QUEUE", "test-queue")
	enableWebSocket := getEnv("ENABLE_WEBSOCKET", "true") == "true"
	enableConsumer := getEnv("ENABLE_CONSUMER", "true") == "true"
	enableHistoryConsumer := getEnv("ENABLE_HISTORY_CONSUMER", "true") == "true"

	// Initialize Messaging
	msgService := messaging.Init(redisHost, redisPort, redisUserName, redisPassword, rabbitMQURL, enableWebSocket, enableConsumer, enableHistoryConsumer)

	// Start WebSocket Server if enabled
	if enableWebSocket {
		go msgService.StartWebSocketServer(webSocketPort, queueName)
	}

	// Start Consumer if enabled
	if enableConsumer {
		bufferSize := 100
		go msgService.StartConsumer(queueName, bufferSize, redisHost, redisPort, redisUserName, redisPassword)
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

//f55da0077317b31bc7af5fde7d46f256
//rabbitmq_dev

//rediss://red-cukv0hin91rc73b06iu0:nHT8A1LNngoS7F1y0VilI1Fe9VKqbgX0@singapore-keyvalue.render.com:6379
//rediss://red-cukv0hin91rc73b06iu0:nHT8A1LNngoS7F1y0VilI1Fe9VKqbgX0@singapore-keyvalue.render.com:6379
