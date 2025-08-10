#!/bin/bash

MODEL_PATH="./ggml-model-Q8_0.gguf"
MPROJ_PATH="./mmproj-model-f16.gguf"

# 判断模型文件是否存在，不存在则下载
if [ ! -f "$MODEL_PATH" ]; then
  echo "Downloading gguf model..."
  wget -c https://huggingface.co/openbmb/MiniCPM-V-4-gguf/resolve/main/ggml-model-Q8_0.gguf
fi

if [ ! -f "$MPROJ_PATH" ]; then
  echo "Downloading mmproj model..."
  wget -c https://huggingface.com/openbmb/MiniCPM-V-4-gguf/resolve/main/mmproj-model-f16.gguf
fi

echo "Starting llama-server with local models..."

llama-server -hf "$MODEL_PATH" --port 8001 --mmproj "$MPROJ_PATH" &

sleep 5

echo ""
echo "启动成功！"
echo "base_url: http://localhost:8001"
echo "api_key: sys_no_key"
echo "model: minicpm-4v"

wait
