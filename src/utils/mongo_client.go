package utils

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var mongo_logger = NewLogger("mongo_client", "logs/mongo_client.log")

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewClient_mongo 初始化 MongoDB 客户端
func NewClient_mongo(uri string, dbName string) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect to MongoDB failed: %v", err)
	}

	// Ping 验证连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping MongoDB failed: %v", err)
	}

	db := client.Database(dbName)
	mongo_logger.Printf("✅ MongoDB connected: %s (db: %s)", uri, dbName)

	return &MongoClient{
		Client:   client,
		Database: db,
	}, nil
}
