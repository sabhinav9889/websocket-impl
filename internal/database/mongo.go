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

func (db *MongoDB) SaveMessage(userID string, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.collection.InsertOne(ctx, bson.M{
		"userID":  userID,
		"message": message,
		"created": time.Now(),
	})
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

