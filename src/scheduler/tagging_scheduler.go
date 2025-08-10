package scheduler

import (
	"bytes"
	"context"
	"dataset_platform/database/models"
	"dataset_platform/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var TaggingLog = utils.NewLogger("tagger", "logs/tagging.log")

func StartTaggingScheduler(mongoClient *utils.MongoClient, s3Client *utils.S3_Client) {
	go func() {
		for {
			processNextDataset(mongoClient, s3Client)
			time.Sleep(1 * time.Second)
		}
	}()
}

func extToFormat(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if ext == "jpg" {
		return "jpeg"
	}
	return ext
}
func processNextDataset(mongoClient *utils.MongoClient, s3Client *utils.S3_Client) {
	collection := mongoClient.Database.Collection("dataset_records")

	var task models.DatasetRecord
	err := collection.FindOneAndUpdate(
		context.TODO(),
		bson.M{"is_labeled": 0},
		bson.M{"$set": bson.M{"is_labeled": 2}},
	).Decode(&task)
	if err != nil {
		// 没有任务或查询错误
		return
	}

	TaggingLog.Println("开始处理数据集:", task.Name)

	zipPath := filepath.Join(os.TempDir(), task.Name+".zip")
	defer os.Remove(zipPath)

	if err := s3Client.DownloadFile(context.TODO(), task.S3Key, zipPath); err != nil {
		TaggingLog.Println("❌ ZIP 下载失败:", err)
		updateStatus(collection, task.ID, -1)
		return
	}

	unzipDir := filepath.Join(os.TempDir(), "unzipped_"+task.Name)
	defer os.RemoveAll(unzipDir)

	if err := utils.Unzip(zipPath, unzipDir); err != nil {
		TaggingLog.Println("❌ 解压失败:", err)
		updateStatus(collection, task.ID, -1)
		return
	}

	// 输出目录
	outputDir := filepath.Join(os.TempDir(), "processed_"+task.Name)
	defer os.RemoveAll(outputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		TaggingLog.Println("❌ 创建输出目录失败:", err)
		updateStatus(collection, task.ID, -1)
		return
	}

	// 遍历并重命名 + 收集标签
	labels := make(map[string]string)
	index := 1

	err = filepath.Walk(unzipDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			TaggingLog.Println("⚠️ 访问文件错误:", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			TaggingLog.Println("⚠️ 跳过非图片文件:", filepath.Base(path))
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			TaggingLog.Println("⚠️ 读取图片失败，删除文件:", path, err)
			if delErr := os.Remove(path); delErr != nil {
				TaggingLog.Println("⚠️ 删除失败:", path, delErr)
			}
			return nil
		}

		newName := fmt.Sprintf("%05d%s", index, ext)
		newPath := filepath.Join(outputDir, newName)
		if err := os.WriteFile(newPath, data, 0644); err != nil {
			TaggingLog.Println("⚠️ 写入新文件失败:", newPath, err)
			return nil
		}

		caption, err := callTagAPIAndGetCaption(newPath, task.Category, extToFormat(ext))
		if err != nil {
			TaggingLog.Println("❌ 调用标签接口失败:", newPath, err)
			if strings.Contains(err.Error(), "base64 解码失败") || strings.Contains(err.Error(), "cannot identify image file") {
				TaggingLog.Println("⚠️ 图片解码失败，删除文件:", newPath)
				if delErr := os.Remove(newPath); delErr != nil {
					TaggingLog.Println("⚠️ 删除失败:", newPath, delErr)
				}
			}
		} else if caption != "" {
			labels[newName] = caption //strings.Split(caption, ",")
		} else {
			TaggingLog.Println("⚠️ 空标签，跳过记录:", newPath)
		}

		index++
		return nil
	})

	if err != nil || len(labels) == 0 {
		TaggingLog.Println("❌ 没有有效图片或标签失败:", err)
		updateStatus(collection, task.ID, -1)
		return
	}

	// 写入 labels.json
	labelJsonPath := filepath.Join(outputDir, "labels.json")
	labelData, _ := json.MarshalIndent(labels, "", "  ")
	if err := os.WriteFile(labelJsonPath, labelData, 0644); err != nil {
		TaggingLog.Println("❌ 写入标签 JSON 失败:", err)
		updateStatus(collection, task.ID, -1)
		return
	}

	// 打包 zip
	newZipPath := filepath.Join(os.TempDir(), task.Name+"_tagged.zip")
	defer os.Remove(newZipPath)

	if err := utils.Zip(outputDir, newZipPath); err != nil {
		TaggingLog.Println("❌ 打包失败:", err)
		updateStatus(collection, task.ID, -1)
		return
	}

	// 上传覆盖
	if err := s3Client.UploadFile(context.TODO(), newZipPath, task.S3Key); err != nil {
		TaggingLog.Println("❌ 上传失败:", err)
		updateStatus(collection, task.ID, -1)
		return
	}

	updateStatus(collection, task.ID, 1)
	TaggingLog.Println("✅ 完成数据集:", task.Name)
}

func updateStatus(collection *mongo.Collection, id primitive.ObjectID, status int) {
	collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"is_labeled": status}},
	)
}

func callTagAPIAndGetCaption(imagePath string, category string, format string) (string, error) {
	tagAPIURL := os.Getenv("TAG_API_URL")
	if tagAPIURL == "" {
		tagAPIURL = "http://localhost:6004"
	}
	// 读取图片文件内容
	data, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("读取图片失败: %w", err)
	}

	// 转base64字符串
	encoded := base64.StdEncoding.EncodeToString(data)
	payload := fmt.Sprintf(`{
		"image_base64": "%s",
		"category": "%s",
		"format" : "%s"

	}`, encoded, category, format)
	resp, err := http.Post(tagAPIURL+"/tag_image", "application/json", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应体，提取 caption
	type ApiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Caption string `json:"caption"`
		} `json:"data"`
	}

	var result ApiResp
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("API 返回错误: %s", result.Msg)
	}

	return result.Data.Caption, nil
}
