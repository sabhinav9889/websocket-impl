// consumer/message_consumer.go
package consumer

import (
	"context"
	"encoding/json"
	"websocket-messaging/internal/database"
	"websocket-messaging/internal/models"
	"websocket-messaging/internal/redis"

	log "github.com/sirupsen/logrus"
)

type MessageConsumer struct {
	RedisClient redis.RedisClient
	MogDb       *database.MongoDB
}

// NewMessageConsumer creates a new instance of MessageConsumer
func GetMessageConsumer(redisClient redis.RedisClient, uri, dbName, collectionName string) *MessageConsumer {
	return &MessageConsumer{redisClient, database.NewDatabase(uri, dbName, collectionName)}
}

func getDbMessage(messageData models.Message) database.ChatMessage {
	var message database.ChatMessage
	message.GroupID = messageData.GroupID
	message.GroupName = messageData.GroupName
	message.MessageId = messageData.MessageId
	message.MessageBody = messageData.Content
	message.ReceiverID = messageData.ReceiverID
	message.SenderID = messageData.SenderID
	// message.TimeStamp = messageData.TimeStamp
	message.DeliveryStatus = messageData.DeliveryStatus
	message.MessageType = messageData.Type
	message.ConversationID = messageData.ConversationID
	message.Status = messageData.Status
	return message
}

func (mc *MessageConsumer) ProcessMessage(ctx context.Context, message string) (bool, error) {
	var messageData models.Message
	err := json.Unmarshal([]byte(message), &messageData)
	if err != nil {
		log.Info("Failed to unmarshal message:", err)
		return false, err
	}
	serverId, err := mc.RedisClient.GetUserServer(messageData.ReceiverID)
	if err != nil {
		log.Info("User is offline : ", err)
		messageData.DeliveryStatus = "sent"
	} else {
		messageData.DeliveryStatus = "delivered"
		go mc.RedisClient.Publish(serverId, message)
	}
	if messageData.Status == "edited" {
		message := getDbMessage(messageData)
		err := mc.MogDb.UpdateMessage(message)
		if err != nil {
			log.Info("Failed to update message in DB:", err)
		}
	} else {
		message := getDbMessage(messageData)
		err := mc.MogDb.SaveMessage(message)
		if err != nil {
			log.Info("Failed to save message to DB:", err)
		}
	}
	return true, nil
}

func (mc *MessageConsumer) ProcessBulkMessage(ctx context.Context, messages []string) (bool, error) {
	log.Info("Processing bulk messages:", messages)
	return true, nil
}

func (mc *MessageConsumer) GetConsumerName() string {
	return "MessageConsumer"
}
