package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoURI     = "mongodb://development:testpassword@localhost:27017"
	databaseName = "Ecommerce"
	connectTimeout = 10 * time.Second
)

// DBConnect initializes and returns a MongoDB client.
func DBConnect() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("MongoDB connection error: %v", err)
	}

	// Ping the database to verify the connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	fmt.Println("✅ Successfully connected to MongoDB")
	return client
}

// Global MongoDB client instance
var Client *mongo.Client = DBConnect()

// GetCollection returns a handle to a MongoDB collection in the "Ecommerce" database.
func GetCollection(collectionName string) *mongo.Collection {
	return Client.Database(databaseName).Collection(collectionName)
}

// Optional convenience wrappers for specific collections (if needed)
func UserCollection() *mongo.Collection {
	return GetCollection("Users")
}

func ProductCollection() *mongo.Collection {
	return GetCollection("Products")
}
