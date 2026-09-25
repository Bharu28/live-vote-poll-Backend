package config

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var MongoClient *mongo.Client
var RedisClient *redis.Client

func ConnectDatabase() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// MongoDB
	mongodbURI := os.Getenv("MONGODB_URI")

	if mongodbURI == "" {
		log.Fatal("MONGODB_URI is not set")
	}

	MongoClient, err = mongo.Connect(
		options.Client().
			ApplyURI(mongodbURI).
			SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}).
			SetConnectTimeout(10 * time.Second).
			SetServerSelectionTimeout(30 * time.Second),
	)

	if err != nil {
		log.Fatal("MongoDB connection failed:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = MongoClient.Ping(ctx, nil)

	if err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	log.Println("MongoDB connected successfully!")

	// Redis is optional so the API can run without realtime streaming.
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" && os.Getenv("PORT") == "" {
		redisURL = "redis://localhost:6379"
	}
	if redisURL == "" {
		log.Println("Warning: REDIS_URL is not set; realtime streaming is disabled")
		return
	}

	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Invalid REDIS_URL:", err)
	}
	RedisClient = redis.NewClient(redisOptions)

	redisCtx, redisCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer redisCancel()

	_, err = RedisClient.Ping(redisCtx).Result()

	if err != nil {
		RedisClient.Close()
		RedisClient = nil
		log.Println("Warning: Redis connection failed; realtime streaming is disabled:", err)
		return
	}

	log.Println("Redis connected successfully!")
}
