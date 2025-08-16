
#================================= 初始化 client ===================================
from openai import OpenAI
import base64
from PIL import Image
import io
from utils.config import vlm_config
import json
from dataclasses import dataclass
#使用兼容openai的VLM 进行图片打标
#print(vlm_config)
client = OpenAI(
    api_key = vlm_config['api_key'],
    base_url= vlm_config['base_url']
)
MODEL_NAME= vlm_config['model_name']
SYSTEM_PROMPT = vlm_config['system_prompt']
#================================================================================



#============================  提示词  ===========================================


USER_QUERY="""According to the content of this picture, write the flux prompt word,Finally Return as json. """




#================================================================================


def resize_image(image_path, max_size=512):
    """等比例缩放图片，使最大边 <= max_size"""
    with Image.open(image_path) as img:
        w, h = img.size
        scale = min(max_size / w, max_size / h) 
        if scale < 1:  
            new_w, new_h = int(w * scale), int(h * scale)
            img = img.resize((new_w, new_h), Image.LANCZOS)
        buffer = io.BytesIO()
        img.save(buffer, format="JPEG", quality=90)
        return buffer.getvalue()

def encode_image(image_path, max_size=512):
    """图片缩放+ jpeg压缩 + Base64 编码"""
    img_bytes = resize_image(image_path, max_size)
    return base64.b64encode(img_bytes).decode('utf-8')


def resize_base64_image(base64_str, max_size=512):
    """
    接受Base64编码的图片字符串，等比例缩放图片（最大边不超过max_size），
    并压缩成JPEG格式，返回新的Base64字符串（不带头部 'data:image...'）。
    """
    # 解码Base64字符串成bytes
    image_data = base64.b64decode(base64_str)
    
    # 用BytesIO读取成PIL Image对象
    with Image.open(io.BytesIO(image_data)) as img:
        w, h = img.size
        scale = min(max_size / w, max_size / h)
        if scale < 1:
            new_w, new_h = int(w * scale), int(h * scale)
            img = img.resize((new_w, new_h), Image.LANCZOS)
        
        # 保存成JPEG到内存
        buffer = io.BytesIO()
        img.save(buffer, format='JPEG', quality=90)
        resized_bytes = buffer.getvalue()
    
    # 再编码成Base64字符串
    resized_base64 = base64.b64encode(resized_bytes).decode('utf-8')
    return resized_base64

@dataclass
class TaggingResult:
    caption:str = ""
    flag:bool = True
    model:str = "VLM"
from dataclasses import dataclass

@dataclass
class FLUX_Options:
    image_base64: str = ""

def auto_tagger(options: FLUX_Options):
    try:
        base64_image = options.image_base64
        base64_image = resize_base64_image(base64_image)
        response = client.chat.completions.create(
            model=MODEL_NAME,
            messages=[
                {"role":"system","content":SYSTEM_PROMPT},
                {"role":"user","content":[
                    {"type":"text","text":USER_QUERY},
                    {"type":"image_url","image_url":{
                        "url": f"data:image/jpeg;base64,{base64_image}"
                    }}
                ]}
            ],
            response_format={ "type": "json_object" },
        )
        json_data = json.loads(response.choices[0].message.content)
        caption = json_data.get("final_prompt", "")
        if caption == "":
            return TaggingResult(caption="", flag=False)
        return TaggingResult(caption=caption, flag=True)
    except Exception as e:
        print("⚠️ Error in use VLM to generate prompt: ", e)
        return TaggingResult(caption="", flag=False)


if __name__ == '__main__':
    # 假设这里是Base64字符串，而不是文件路径
    with open("./test.jpeg", "rb") as f:
        img_bytes = f.read()
    base64_str = base64.b64encode(img_bytes).decode('utf-8')

    options = FLUX_Options(image_base64=base64_str)
    print(auto_tagger(options))
