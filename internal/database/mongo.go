// mongo.go
package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatMessage struct {
	SenderID         string    `bson:"senderID"`
	ReceiverID       string    `bson:"receiverID"`
	MessageBody      string    `bson:"messageBody"`
	MessageType      string    `bson:"messageType"` // "text", "image", "video", etc.
	Status           string    `bson:"status"`      // "sent", "delivered", "read"
	ReceivedAt       time.Time `bson:"receivedAt"`
	CreatedAt        time.Time `bson:"createdAt"`
	ConversationID   string    `bson:"conversationID"`
	ReplyToMessageID *string   `bson:"replyToMessageID,omitempty"` // Nullable for replies
}

type MongoDB struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewDatabase(uri, dbName, collectionName string) *MongoDB {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to create MongoDB client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	return &MongoDB{
		client:     client,
		collection: client.Database(dbName).Collection(collectionName),
	}
}

func (db *MongoDB) SaveMessage(chatMessage ChatMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	messageBody := bson.M{
		"senderID":       chatMessage.SenderID,       // ID of the sender
		"receiverID":     chatMessage.ReceiverID,     // ID of the receiver
		"content":        chatMessage.MessageBody,    // Message text or file URL
		"messageType":    chatMessage.MessageType,    // "text", "image", "video", etc.
		"status":         chatMessage.Status,         // "sent", "delivered", "read"
		"receivedAt":     chatMessage.ReceivedAt,     // When the message was received
		"createdAt":      time.Now(),                 // When the message was stored
		"conversationID": chatMessage.ConversationID, // Unique conversation identifier
	}
	_, err := db.collection.InsertOne(ctx, messageBody)
	return err
}

func (db *MongoDB) GetPendingMessages(userID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var results []string
	cursor, err := db.collection.Find(ctx, bson.M{"userID": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var message struct {
			Message string `bson:"message"`
		}
		if err := cursor.Decode(&message); err != nil {
			return nil, err
		}
		results = append(results, message.Message)
	}
	return results, nil
}

func (db *MongoDB) DeletePendingMessages(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.collection.DeleteMany(ctx, bson.M{"userID": userID})
	return err
}
