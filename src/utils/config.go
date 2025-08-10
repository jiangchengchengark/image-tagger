package utils

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

// S3Config 结构体用于映射config.yaml中的s3配置
type S3Config struct {
	EndpointURL string `yaml:"endpoint_url"`
	RegionName  string `yaml:"region_name"`
	AccessKey   string `yaml:"access_key"`
	SecretKey   string `yaml:"secret_key"`
	Bucket      string `yaml:"bucket"`
}

// MongoConfig 结构体用于映射config.yaml中的mongo配置
type MongoConfig struct {
	Uri      string `yaml:"uri"`
	Database string `yaml:"database"`
}

// RedisConfig 结构体用于映射config.yaml中的redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
}

// Config 是全局配置结构体
type Config struct {
	S3    S3Config    `yaml:"s3"`
	Mongo MongoConfig `yaml:"mongo"`
	Redis RedisConfig `yaml:"redis"`
}

var AppConfig *Config

var configLogger *log.Logger

// LoadConfig  从指定路径读取config.yaml文件，并解析到全局AppConfig
func LoadConfig(path string) {
	configLogger = NewLogger("config", "logs/config.log")
	file, err := os.Open(path)
	if err != nil {
		configLogger.Fatalf("❌ open config file failed, err: %v", err)

	}
	defer file.Close()
	cfg := &Config{}
	// yaml文件中的内容读取并解析到go的结构体
	if err := yaml.NewDecoder(file).Decode(cfg); err != nil {
		configLogger.Fatalf("❌ parse config file failed, err: %v", err)

	}

	AppConfig = cfg
	configLogger.Println("✅ load config file success")

}
