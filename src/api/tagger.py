from fastapi import FastAPI
from pydantic import BaseModel
from typing import Optional
from PIL import Image
import base64
import io

from utils.sdxl_tagger import auto_tagger as sdxl_tagger 
from utils.sdxl_tagger import SDXL_Options, load_model_and_labels
from utils.flux_tagger import auto_tagger as flux_tagger 
from utils.flux_tagger import FLUX_Options


app = FastAPI()

class ImageInput(BaseModel):
    image_base64: str
    category: str = "sdxl"
    format: Optional[str] = "jpeg"


class APIResponse(BaseModel):
    code: int
    msg: str
    data: dict


@app.on_event("startup")
def preload():
    load_model_and_labels("vit")
    print("preload done with vit model")

@app.post("/tag_image")
async def tag_image(payload: ImageInput):
    try:
        # 解码base64
        image_data = base64.b64decode(payload.image_base64)
        image = Image.open(io.BytesIO(image_data))
        image_format = payload.format or "jpeg"
    except Exception as e:
        return APIResponse(code=-1, msg="图片解码失败: " + str(e), data={})

    try:
        if payload.category == "sdxl":
            opts = SDXL_Options(image_base64=payload.image_base64)
            result = sdxl_tagger(opts)
        elif payload.category == "flux":
            opts = FLUX_Options(image_base64=payload.image_base64)
            result = flux_tagger(opts)
        else:
            opts = SDXL_Options(image_base64=payload.image_base64)
            result = sdxl_tagger(opts)

        if not result.flag and result.model == "VLM":
            opts = SDXL_Options(image_base64=payload.image_base64)
            result = sdxl_tagger(opts)

        if not result.flag:
            return APIResponse(code=-1, msg=f"{result.model}模型处理失败", data={})

        return APIResponse(code=0, msg="success", data={"caption": result.caption})

    except Exception as e:
        return APIResponse(code=-1, msg="模型处理失败: " + str(e), data={})


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=6004)