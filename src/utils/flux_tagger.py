
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
You are a professional AI painting prompt engineer, skilled at converting image content into high-quality Flux generative model prompts.
Now, based on the input image, you are asked to infer its original generated prompt (Flux Prompt).
The description must include information such as the main subject, scene, details, style, lighting, and color, while retaining the stylistic terms.
The output format must be as follows:
{
"main_subject": "<Short English sentence, such as red-haired girl in school uniform>",
"details": "<Long English sentence describing pose, scene, details, emotion, etc.>",
"style": "<Flux style keywords, such as hyper realistic, soft lighting, cinematic>",
"final_prompt": "<Combining main_subject + details + style to form a prompt that can be used directly in Flux>"
}
Notes:
1. Use concise English descriptions that conform to Stable Diffusion / Flux conventions.
2. Avoid using terms unrelated to AI painting, such as "in the image" and "photo of".
3. Do not invent non-existent elements.
4. If the image style is illustration, painting, photography, etc., be sure to include the style.
Example 1:
Image: A blonde girl reading a book in a cafe, sunlight streaming in, Japanese illustration style
Output:
{
"main_subject": "blonde girl reading book in cafe",
"details": "sunlight streaming through window, wooden furniture, soft warm colors",
"style": "anime style, Makoto Shinkai inspired, soft light",
"final_prompt": "blonde girl reading book in cafe, sunlight streaming through window, wooden furniture, soft warm colors, anime style, Makoto Shinkai inspired, soft light"
}

Example 2:
Image: A waterfall in the mountains, mist, realistic photography
Output:
{
"main_subject": "majestic waterfall in the mountains",
"details": "mist rising, lush green forest, flowing water, rocks",
"style": "ultra realistic photography, 8k, HDR",
"final_prompt": "majestic waterfall in the mountains, mist rising, lush green forest, flowing water, rocks, ultra realistic photography, 8k, HDR"
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
