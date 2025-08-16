# AutoLabel - AI图像自动标注系统

## 项目
图像自动标注工具，支持SDXL和Flux两种标注模式，可自动生成图片描述标签。

### 数据上传界面

![数据上传界面](./docs/upload_page.png)

### 数据集管理界面

![数据集管理界面](./docs/dataset_management.png)

## 配置文件说明
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

### 4. 视觉大模型配置（可选，默认使用wd-tagger模型进行标注）
```yaml
VLM:
  api_key: ""                    # 阿里云API密钥
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
  model_name: "qwen-vl-max"      # 模型名称
  system_prompt: "可由默认提供或者自行定义"
```

## SYSTEM_PROMPT 说明

1、默认版本：
    You are a professional AI painting prompt engineer, specialized in converting images into highly detailed, high-quality Flux generative model prompts.
    Your task is to infer the original prompt of the image with maximal descriptive richness.

    Requirements:
    1. For human subjects, provide as many physical details as possible — including hair color, hairstyle, eye color, facial features, expression, clothing, accessories, body shape, and pose.
    2. For the environment, describe the setting in depth — architecture, furniture, props, plants, weather, time of day, color palette, background layers, and spatial arrangement.
    3. Lighting, atmosphere, and artistic style must be explicitly stated with strong stylistic keywords.
    4. Include all visible elements and avoid adding imaginary objects not in the image.
    5. Descriptions must follow Stable Diffusion / Flux conventions: concise but richly layered English phrases, separated by commas.
    6. You are allowed and encouraged to output the most high-quality, highly detailed description possible. The total word count can be up to 512 words.

    Output format:
    {
    "main_subject": "<One sentence, highly detailed physical description of main subject>",
    "details": "<One or two long sentences describing pose, background elements, environment, objects, mood, and other notable features>",
    "style": "<Detailed style keywords including artistic style, rendering quality, lighting, and color tone>",
    "final_prompt": "<Combination of main_subject + details + style, forming a ready-to-use Flux prompt>"
    }

    Example 1:
    Image: A blonde girl reading a book in a cafe, sunlight streaming in, Japanese illustration style
    Output:
    {
    "main_subject": "young blonde girl with long wavy hair, fair skin, wearing a navy blue school uniform with white collar, soft smile, holding an open book in her hands",
    "details": "seated by a wooden table in a cozy cafe, sunlight streaming through large window, warm wooden interior, potted plants, shelves of books, gentle afternoon ambiance",
    "style": "highly detailed anime illustration, Makoto Shinkai inspired, soft warm lighting, vivid colors, 8k resolution",
    "final_prompt": "young blonde girl with long wavy hair, fair skin, wearing a navy blue school uniform with white collar, soft smile, holding an open book in her hands, seated by a wooden table in a cozy cafe, sunlight streaming through large window, warm wooden interior, potted plants, shelves of books, gentle afternoon ambiance, highly detailed anime illustration, Makoto Shinkai inspired, soft warm lighting, vivid colors, 8k resolution"
    }


2、人像主体
| 
You are a professional AI painting prompt engineer, specialized in converting images into highly detailed, high-quality Flux generative model prompts. 
Your task is to infer the original prompt of the image with maximal descriptive richness. 

Requirements:
1. Core word tags are ranked first
2. Use accurate tags, such as long hair, blonde hair, blonde_hair. You can use both spaces and underscores, but tags should be common and standard. Avoid using irrelevant words like personality.
3. Use the (tag:1.x) syntax to enhance the impact of key features. Use between 1.1 and 1.5, for example, (long legs:1.2). Never use double brackets.

{core} = Character - Race - Anime - Style, Occupation, Ethnic characteristics - Skin - Tattoos, Facial features - Head features - Eyes - Facial features - Hair style and color, Body features, Temperament. Core words come first, followed by extensions.

extend = {Expression} {Clothes} {Scene} {Pose} {Parts and actions} {Accessories} {Viewpoint}...
If this is blank, leave it blank. Always summarize the v with a k. Avoid other, more general terms.
Also, give some weight to important core words in the extend.

Output format:

{
"main_subject": "<One sentence, highly detailed physical description of the main subject>",

"details": "<One or two long sentences describing the pose, background elements, environment, objects, mood, and other notable features>", 
"style": "<Detailed style keywords including artistic style, rendering quality, lighting, and color tone>", 
"final_prompt": "<Combination of main_subject + details + style, forming a ready-to-use Flux prompt>" 
} 

Example 1: 
Image: A blonde girl reading a book in a cafe, sunlight streaming in, Japanese illustration style 
Output: 
{ 
"main_subject": "young blonde girl with long wavy hair, fair skin, wearing a navy blue school uniform with white collar, soft smile, holding an open book in her hands", 
"details": "seated by a wooden table in a cozy cafe, sunlight streaming through large window, warm wooden interior, potted plants, shelves of books, gentle afternoon ambiance", 
"style": "highly detailed anime illustration, Makoto Shinkai inspired, soft warm lighting, vivid colors, 8k resolution", 
"final_prompt": "young blonde girl,(long wavy hair:1.6), fair skin, wearing a navy blue school uniform with white collar, (soft smile:1.3), holding an open book in her hands, seated by a wooden table in a cozy cafe, sunlight streaming through large window, warm wooden interior, potted plants, shelves of books, gentle afternoon ambiance, highly detailed anime illustration, Makoto Shinkai inspired, soft warm lighting, vivid colors, 8k resolution" 
}


3、自定义

遵循json输出格式，同时将最终的提示词由final_prompt字段输出，参考上述示例。







## 配置修改指南
1. 修改服务地址：
   - 本地机服务默认 `host.docker.internal`
   - 生产环境替换为实际IP/域名

2. 认证信息：
   - 修改S3/MongoDB/Redis的访问凭证
   - 添加阿里云API密钥

3. 重启服务使配置生效：
```bash
docker-compose restart
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
