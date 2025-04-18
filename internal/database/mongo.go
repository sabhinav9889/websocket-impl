// mongo.go
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatMessage struct {
	MessageId        string    `bson:"messageId"`
	SenderID         string    `bson:"senderID"`
	GroupID          string    `bson:"groupID"` // Nullable for group messages
	GroupName        string    `bson:"groupName"`
	ReceiverID       string    `bson:"receiverID"`
	MessageBody      string    `bson:"messageBody"`
	MessageType      string    `bson:"messageType"`    // "text", "image", "video", etc.
	Status           string    `bson:"status"`         // "online", "typing", "edited"
	DeliveryStatus   string    `bson:"deliveryStatus"` // "sent", "delivered", "read"
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
	fmt.Println("Connecting to MongoDB...", uri, dbName, collectionName)
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
		"messageId":      chatMessage.MessageId,      // Unique message identifier
		"groupID":        chatMessage.GroupID,        // Nullable for group messages
		"groupName":      chatMessage.GroupName,      // Nullable for group messages
		"senderID":       chatMessage.SenderID,       // ID of the sender
		"receiverID":     chatMessage.ReceiverID,     // ID of the receiver
		"messageBody":    chatMessage.MessageBody,    // Message text or file URL
		"messageType":    chatMessage.MessageType,    // "text", "image", "video", etc.
		"status":         chatMessage.Status,         // "online", "typing", "edited"
		"deliveryStatus": chatMessage.DeliveryStatus, // "sent", "delivered", "read"
		"receivedAt":     chatMessage.ReceivedAt,     // When the message was received
		"createdAt":      time.Now(),                 // When the message was stored
		"conversationID": chatMessage.ConversationID, // Unique conversation identifier
	}
	_, err := db.collection.InsertOne(ctx, messageBody)
	return err
}

func (db *MongoDB) UpdateMessage(chatMessage ChatMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	update := bson.M{
		"$set": bson.M{
			"messageBody":    chatMessage.MessageBody,
			"status":         chatMessage.Status,
			"deliveryStatus": chatMessage.DeliveryStatus,
			"receivedAt":     chatMessage.ReceivedAt,
			"createdAt":      time.Now(),
			"messageType":    chatMessage.MessageType,
		},
	}
	filter := bson.M{"messageId": chatMessage.MessageId, "receiverID": chatMessage.ReceiverID}
	res, err := db.collection.UpdateOne(ctx, filter, update)
	fmt.Println("Update result:", res)
	if res.MatchedCount == 0 {
		fmt.Println("No document matched the filter")
	}
	return err
}

func (db *MongoDB) ChangeMessageStatus(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.collection.UpdateMany(ctx, bson.M{"receiverID": userID}, bson.M{"$set": bson.M{"deliveryStatus": "delivered"}})
	return err
}

func (db *MongoDB) GetPendingMessages(userID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var results []string
	cursor, err := db.collection.Find(ctx, bson.M{"receiverID": userID, "deliveryStatus": "sent"})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var results1 []ChatMessage
	if err = cursor.All(ctx, &results1); err != nil {
		fmt.Println("Error decoding cursor:", err)
		return nil, err
	}
	for _, person := range results1 {
		msg, err := json.Marshal(person)
		if err != nil {
			fmt.Println("Error marshaling person:", err)
			return nil, err
		}
		results = append(results, string(msg))
	}
	fmt.Println("Pending messages: 123", results, results1)
	return results, nil
}

func (db *MongoDB) DeletePendingMessages(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.collection.DeleteMany(ctx, bson.M{"userID": userID})
	return err
}
