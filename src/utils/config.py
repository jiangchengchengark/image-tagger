#读取config.yaml文件的VLM部分配置

import yaml 

# 读取config.yaml文件
with open('config.yaml', 'r', encoding='utf-8') as f:
    config = yaml.load(f, Loader=yaml.FullLoader)

# 获取VLM配置
vlm_config = config['VLM']

__all__=[
    'vlm_config'
]