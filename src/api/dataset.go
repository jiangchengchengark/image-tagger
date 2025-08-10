package api

import (
	"context"
	"dataset_platform/database/models"
	"dataset_platform/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const basePrefix = "AI/train/I2M"

var dataset_logger = utils.NewLogger("dataset_server", "logs/dataset_server.log")

var S3_client *utils.S3_Client
var Mongo_client *utils.MongoClient

func init() {
	utils.LoadConfig("config.yaml")
	config := utils.AppConfig

	// S3 init
	new_client_s3, err := utils.NewClient_s3(config.S3.EndpointURL, config.S3.RegionName, config.S3.AccessKey, config.S3.SecretKey, config.S3.Bucket)
	if err != nil {
		fmt.Println("S3 客户端初始化失败: ", err)
		os.Exit(1)
	}
	S3_client = new_client_s3
	dataset_logger.Println("S3 客户端初始化成功")

	// Mongo init
	mongoClient, err := utils.NewClient_mongo(config.Mongo.Uri, config.Mongo.Database)
	if err != nil {
		fmt.Println("MongoDB 客户端初始化失败: ", err)
		os.Exit(1)
	}
	Mongo_client = mongoClient
	dataset_logger.Println("MongoDB 客户端初始化成功")

	// 初始化索引
	collection := Mongo_client.Database.Collection("dataset_records")
	_, err = models.CreateIndexes4DatasetRecord(collection)
	if err != nil {
		fmt.Println("创建索引失败: ", err)
		os.Exit(1)
	}
	dataset_logger.Println("💪 创建索引成功")
}

// 统一响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": data,
	})
}

func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": -1,
		"msg":  msg,
		"data": gin.H{},
	})
}

// 上传数据集 ZIP 文件
func UploadDataset(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		Error(c, "缺少 name 参数")
		return
	}

	// 检查是否已存在
	collection := Mongo_client.Database.Collection("dataset_records")
	var existing models.DatasetRecord
	err := collection.FindOne(context.Background(), bson.M{"name": name}).Decode(&existing)
	if err == nil {
		Error(c, "数据集名称已存在，禁止重复上传")
		return
	}
	if err != mongo.ErrNoDocuments {
		Error(c, "数据库查询失败: "+err.Error())
		return
	}

	category := c.PostForm("category")
	isLabeled := 0
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		Error(c, "文件读取失败")
		return
	}
	defer file.Close()
	dataset_logger.Printf("接收到数据集文件: %s", header.Filename)

	tempPath := filepath.Join(os.TempDir(), header.Filename)
	out, err := os.Create(tempPath)
	if err != nil {
		Error(c, "临时文件创建失败")
		return
	}
	io.Copy(out, file)
	out.Close()

	// 上传到 S3
	s3key := fmt.Sprintf("%s/%s", basePrefix, name+".zip")
	err = S3_client.UploadFile(context.Background(), tempPath, s3key)
	if err != nil {
		Error(c, "S3 上传失败: "+err.Error())
		return
	}

	// 插入 MongoDB
	record := models.DatasetRecord{
		ID:        primitive.NewObjectID(),
		Name:      name,
		S3Key:     s3key,
		Category:  category,
		IsLabeled: isLabeled,
		CreateAt:  time.Now(),
	}
	_, err = collection.InsertOne(context.Background(), record)
	if err != nil {
		Error(c, "数据库插入失败: "+err.Error())
		return
	}
	dataset_logger.Printf("⭕️ 数据集 %s 插入记录保存至 mongodb", name)

	Success(c, gin.H{"name": name})
}

// 获取数据集列表， 支持分类 关键词模糊匹配
func ListDatasets(c *gin.Context) {
	keyword := c.Query("keyword")
	category := c.Query("category") // 新增读取分类参数

	filter := bson.M{}

	if keyword != "" {
		filter["name"] = bson.M{
			"$regex":   keyword,
			"$options": "i", // 忽略大小写
		}
	}

	if category != "" {
		filter["category"] = category // 精确匹配分类
	}

	collection := Mongo_client.Database.Collection("dataset_records")
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		Error(c, "数据库查询失败")
		return
	}
	defer cursor.Close(context.Background())

	var datasets []models.DatasetRecord
	if err = cursor.All(context.Background(), &datasets); err != nil {
		Error(c, "数据解析失败")
		return
	}
	Success(c, gin.H{"list": datasets})
}

// 下载数据集
func DownloadDataset(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		Error(c, "缺少 name 参数")
		return
	}

	s3key := fmt.Sprintf("%s/%s", basePrefix, name+".zip")
	zipPath := filepath.Join(os.TempDir(), name+".zip")
	err := S3_client.DownloadFile(context.Background(), s3key, zipPath)
	if err != nil {
		Error(c, "下载失败: "+err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+name+".zip")
	c.File(zipPath)

	// 异步删除
	go func() { _ = os.Remove(zipPath) }()
}

// 删除数据集
func DeleteDataset(c *gin.Context) {
	// 这里用name参数，前端也可以改成id，取决于你数据结构
	name := c.Query("name")
	if name == "" {
		Error(c, "缺少 name 参数")
		return
	}

	collection := Mongo_client.Database.Collection("dataset_records")

	// 先查询数据库看是否存在，拿到S3Key
	var record models.DatasetRecord
	err := collection.FindOne(context.Background(), bson.M{"name": name}).Decode(&record)
	if err == mongo.ErrNoDocuments {
		Error(c, "数据集不存在")
		return
	} else if err != nil {
		Error(c, "数据库查询失败: "+err.Error())
		return
	}

	// 删除 S3 上对应的文件
	err = S3_client.DeleteFile(context.Background(), record.S3Key)
	if err != nil {
		Error(c, "删除S3文件失败: "+err.Error())
		return
	}

	// 删除MongoDB中的记录
	_, err = collection.DeleteOne(context.Background(), bson.M{"name": name})
	if err != nil {
		Error(c, "删除数据库记录失败: "+err.Error())
		return
	}

	Success(c, gin.H{"msg": "删除成功"})
}

/*

API使用：

上传数据集：
curl -X POST http://localhost:6060/upload_dataset \
  -F "name=test" \
  -F "category=test" \
  -F "file=@./test.zip"




*/
