# 生图打标工具：支持 SDXL 和 Flux 两种形式的图片打标

## 功能特性
- S3 文件上传/下载管理
- 可视化数据集管理界面
- 图片标注 API (WD 或 Visual Model)
- 支持 Docker 容器化部署

## 系统架构
- 前端: React (端口 3000)
- Go 后端: Gin (端口 6060)
- Python API: FastAPI (端口 6004)

## 快速开始

### 1. 使用 Docker Compose 启动
```bash
docker-compose up -d
```

### 2. 访问服务
- 前端界面: http://localhost:3000
- Go API 文档: http://localhost:6060/swagger
- Python API 文档: http://localhost:6004/docs

## API 接口文档

## FastAPI 接口 (端口: 6004)

### POST /tag_image
图片标注接口

请求参数 (JSON):
```json
{
  "image_base64": "base64编码的图片数据",
  "model": "vit (可选, 默认: vit)",
  "gen_threshold": 0.35 (可选, 默认: 0.35),
  "char_threshold": 0.75 (可选, 默认: 0.75),
  "format": "jpeg (可选, 默认: jpeg)"
}
```

成功响应:
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "caption": "生成的标注文本"
  }
}
```

错误响应:
```json
{
  "code": -1,
  "msg": "错误信息",
  "data": {
    "caption": ""
  }
}
```

## Gin 接口 (端口: 6060)

### POST /upload_dataset
上传数据集ZIP文件

请求参数 (form-data):
- name: 数据集名称 (必填)
- category: 数据集分类 (可选)
- file: ZIP文件 (必填)

成功响应:
```json
{
  "message": "上传成功"
}
```

### GET /list_datasets
获取数据集列表

成功响应:
```json
[
  {
    "id": "ObjectID",
    "name": "数据集名称",
    "s3Key": "S3存储路径",
    "category": "分类",
    "isLabeled": 0,
    "createAt": "创建时间"
  }
]
```

### GET /download_dataset
下载数据集ZIP包

请求参数 (query):
- name: 数据集名称 (必填)

响应: 直接返回ZIP文件流
