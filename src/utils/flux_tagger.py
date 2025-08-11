
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

#================================================================================



#============================  提示词  ===========================================

SYSTEM_PROMPT="""
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
"""

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
