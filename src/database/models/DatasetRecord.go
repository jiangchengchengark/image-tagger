// 数据模型
package models

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type DatasetRecord struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	S3Key     string             `bson:"s3_key" json:"s3_key"`
	Name      string             `bson:"name" json:"name"`
	Category  string             `bson:"category" json:"category"`
	IsLabeled int                `bson:"is_labeled" json:"is_labeled"` // 0: 未处理, 2: 处理中, 1: 已完成
	CreateAt  time.Time          `bson:"create_at" json:"create_at"`
}

// 索引重复创建，不影响开销
func CreateIndexes4DatasetRecord(collection *mongo.Collection) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"is_labeled": 1},
			Options: options.Index().SetName("idx_is_labeled"),
		},
		{
			Keys:    map[string]interface{}{"category": 1},
			Options: options.Index().SetName("idx_category"),
		},
		{
			Keys:    map[string]interface{}{"name": 1},
			Options: options.Index().SetName("idx_name"),
		},
	}

	result, err := collection.Indexes().CreateMany(ctx, indexes)
	return result, err
}
