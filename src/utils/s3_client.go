package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var s3_logger = NewLogger("s3_client", "logs/s3_client.log")

// Client 封装S3客户端
type S3_Client struct {
	svc    *s3.Client
	bucket string
}

// NewClient_s3 初始化S3客户端
func NewClient_s3(endpoint, region, accessKey, secretKey, bucket string) (*S3_Client, error) {
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if endpoint != "" {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: region,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("endpoint is empty")
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolver(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config failed: %v", err)
	}

	svc := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // minio 强制路径形式
	})

	ctx := context.Background()

	// 检查 bucket 是否存在
	_, err = svc.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err != nil {
		var nfe *s3types.NotFound
		if errors.As(err, &nfe) {
			// bucket 不存在，创建 bucket
			_, err = svc.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: &bucket,
			})
			if err != nil {
				return nil, fmt.Errorf("create bucket failed: %v", err)
			}
		} else {
			// 其他错误，返回
			return nil, fmt.Errorf("head bucket failed: %v", err)
		}
	}

	return &S3_Client{svc: svc, bucket: bucket}, nil
}

// UploadFile 上传单个文件到S3
func (c *S3_Client) UploadFile(ctx context.Context, localFilePath, s3Key string) error {
	f, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &c.bucket,
		Key:    &s3Key,
		Body:   f,
		ACL:    s3types.ObjectCannedACLPrivate,
	})
	return err
}

// ListObjects 列举指定前缀的文件
func (c *S3_Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(c.svc, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &prefix,
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range output.Contents {
			keys = append(keys, *obj.Key)
		}
	}
	s3_logger.Printf("get downloading list-objects: %v", keys)
	return keys, nil
}

// DownloadFile 从S3下载单个文件到本地指定路径
func (c *S3_Client) DownloadFile(ctx context.Context, s3Key, localPath string) error {
	output, err := c.svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &s3Key,
	})
	if err != nil {
		return err
	}
	defer output.Body.Close()

	// 创建本地文件夹
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	outFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, output.Body)
	return err
}

// DownloadFolder 下载指定前缀的文件夹下所有文件到本地指定路径
func (c *S3_Client) DownloadFolder(ctx context.Context, prefix, localPath string) error {
	// s3_logger.Print("start downloading folder: %s -> %s", prefix, localPath)
	keys, err := c.ListObjects(ctx, prefix)
	if err != nil {
		return err
	}
	for _, key := range keys {
		filename := filepath.Base(key)
		if err := c.DownloadFile(ctx, key, filepath.Join(localPath, filename)); err != nil {
			return err
		}
	}
	return nil
}

// CountFiles 统计目录下所有文件数（递归）
func CountFiles(folderPath string) (int64, error) {
	var count int64
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			atomic.AddInt64(&count, 1)
		}
		return nil
	})
	return count, err
}

// UploadFolder 上传文件夹所有内容到 S3，支持回调显示进度
func (c *S3_Client) UploadFolder(folderPath, prefix string, callback func(localPath, s3Key string)) error {
	var done int64 = 0

	// 先统计文件总数，方便进度展示
	total, err := CountFiles(folderPath)
	if err != nil {
		return fmt.Errorf("统计文件失败: %w", err)
	}

	return filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return err
		}

		s3Key := filepath.ToSlash(filepath.Join(prefix, relPath))

		if err := c.UploadFile(context.TODO(), path, s3Key); err != nil {
			return err
		}

		atomic.AddInt64(&done, 1)

		if callback != nil {
			callback(path, s3Key)
		} else {
			// 默认打印进度
			fmt.Printf("📤 [%d/%d] Uploaded: %s → %s\n", done, total, path, s3Key)
		}

		return nil
	})
}

// 删除方法
// DeleteFile 删除 S3 上的指定文件
func (c *S3_Client) DeleteFile(ctx context.Context, s3Key string) error {
	_, err := c.svc.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &c.bucket,
		Key:    &s3Key,
	})
	if err != nil {
		return err
	}

	// 等待删除完成（可选）
	waiter := s3.NewObjectNotExistsWaiter(c.svc)
	err = waiter.Wait(ctx, &s3.HeadObjectInput{
		Bucket: &c.bucket,
		Key:    &s3Key,
	}, 5*time.Minute)
	if err != nil {
		return err
	}

	s3_logger.Printf("Deleted S3 file: %s", s3Key)
	return nil
}
