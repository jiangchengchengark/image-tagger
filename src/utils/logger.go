package utils

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// 默认全局日志
var Logger *log.Logger

// 只初始化一次
func init() {
	Logger = NewLogger("app", "logs/app.log") //app放核心交互逻辑
	Logger.Printf("初始全局日志记录器")
}

// NewLogger 创建一个带前缀的日志记录器
func NewLogger(name string, logPath string) *log.Logger {
	// 自动创建目录
	dir := filepath.Dir(logPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("❌ 创建日志目录失败: %v", err)
		}
	}

	// 打开文件
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("❌ 无法打开日志文件 %s: %v", logPath, err)
	}

	// 控制台 + 文件输出
	multiWriter := io.MultiWriter(os.Stdout, file)

	// 创建带前缀的日志对象
	prefix := "[" + name + "] "
	return log.New(multiWriter, prefix, log.LstdFlags|log.Lshortfile)
}
