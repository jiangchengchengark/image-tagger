// 主代码
package main

/*
upload_dataset: 上传zip格式的数据集
curl -X POST http://localhost:6060/upload_dataset \  -F "name=cat_dataset" \
  -F "file=@./test.zip"


{"message":"上传成功"}

download_dataset: 下载指定数据集
curl -X GET "http://localhost:6060/download_dataset?name=cat_dataset" -o download_dataset.zip

直接解压会变成图片，没有文件夹


*/
import (
	"dataset_platform/api"
	"dataset_platform/scheduler"
	"dataset_platform/utils"

	"github.com/gin-gonic/gin"
)

func main() {

	//创建logger
	logger := utils.NewLogger("main", "logs/main.log")
	scheduler.StartTaggingScheduler(api.Mongo_client, api.S3_client)
	//创建路由
	route := gin.Default()
	route.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	route.POST("upload_dataset", api.UploadDataset)
	route.GET("download_dataset", api.DownloadDataset)
	route.GET("list_datasets", api.ListDatasets)
	route.DELETE("/delete_dataset", api.DeleteDataset)
	logger.Println("✅ server start success")
	route.Run(":6060")

}
