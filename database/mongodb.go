package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ConnectDB() *mongo.Client {

	uri := os.Getenv("MONGODB_URI")

	if uri == "" {
		log.Fatal("MONGODB_URI is not set")
	}

	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(30 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection failed:", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	log.Println("MongoDB connected successfully!")

	return client
}