# AutoLabel - AI图像自动标注系统

## 目录
- [项目概述](#项目概述)
- [界面预览](#界面预览)
- [配置说明](#配置说明)
  - [S3存储配置](#1-s3存储配置)
  - [MongoDB配置](#2-mongodb配置)
  - [Redis配置](#3-redis配置)
  - [视觉大模型配置](#4-视觉大模型配置可选默认使用wd-tagger模型进行标注)
- [快速开始](#快速开始)
- [项目结构](#项目结构)
- [SYSTEM_PROMPT说明](#system_prompt-说明)

## 项目概述
图像自动标注工具，支持SDXL和Flux两种标注模式，可自动生成图片描述标签。

## 界面预览
| 功能界面 | 预览 |
|---------|------|
| 数据上传界面 | ![数据上传界面](./docs/upload_page.png) |
| 数据集管理界面 | ![数据集管理界面](./docs/dataset_management.png) |

## 配置说明
配置文件位于`config.yaml`，主要配置如下：

### 1. S3存储配置
```yaml
s3:
  endpoint_url: "http://host.docker.internal:9000" # MinIO地址
  region_name: "us-east-1"
  access_key: "minioadmin"       # 访问密钥
  secret_key: "minioadmin"       # 私有密钥
  bucket: "ai-dataset"           # 存储桶名称
```

### 2. MongoDB配置
```yaml
mongo:
  uri: "mongodb://host.docker.internal:27017" # MongoDB地址
  database: "ai-dataset"         # 数据库名称
```

### 3. Redis配置
```yaml
redis:
  host: "host.docker.internal"   # Redis地址
  port: "6379"                   # Redis端口
  password: ""                   # 认证密码
  database: 0                    # 数据库索引
```

### 4. 视觉大模型配置（可选）
```yaml
VLM:
  api_key: ""                    # 阿里云API密钥
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
  model_name: "qwen-vl-max"      # 模型名称
  system_prompt: "可由默认提供或者自行定义"
```

## 快速开始
```bash
# 启动所有服务
docker-compose up --build -d

# 访问服务
Web界面: http://localhost:3000
API文档: http://localhost:6060/docs
```

## 项目结构
```
├── config.yaml       # 主配置文件
├── api/              # API服务
├── web/              # 前端界面
└── utils/            # 模型实现
```

## SYSTEM_PROMPT 说明

系统提供了三种不同的PROMPT模板，可根据需求选择使用：

1. [默认版本](./SYSTEM_PROMPT.md#1-默认版本) - 通用图像描述模板
2. [人像主体版本](./SYSTEM_PROMPT.md#2-人像主体版本) - 专门针对人像优化的模板
3. [自定义版本](./SYSTEM_PROMPT.md#3-自定义版本) - 可自行扩展的模板结构

详细说明和示例请查看[SYSTEM_PROMPT文档](./SYSTEM_PROMPT.md)
