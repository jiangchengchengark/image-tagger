""" 
该脚本 提供自动打标方法

输入一张图片，返回标注text

"""
import base64
import io

from dataclasses import dataclass, field
#=======================  wd 模型配置  ===========================

MODEL_REPO_MAP = {
    "vit": "SmilingWolf/wd-vit-tagger-v3",
    "swinv2": "SmilingWolf/wd-swinv2-tagger-v3",
    "convnext": "SmilingWolf/wd-convnext-tagger-v3",
}

from PIL import Image
#对RGBA的图像转化为RGB
def pil_ensure_rgb(image: Image.Image) -> Image.Image:
    # convert to RGB/RGBA if not already (deals with palette images etc.)
    if image.mode not in ["RGB", "RGBA"]:
        image = image.convert("RGBA") if "transparency" in image.info else image.convert("RGB")
    # convert RGBA to RGB with white background
    if image.mode == "RGBA":
        canvas = Image.new("RGBA", image.size, (255, 255, 255))
        canvas.alpha_composite(image)
        image = canvas.convert("RGB")
    return image

#将图像填充至正方形，背景白色，保持原图中央
def pil_pad_square(image: Image.Image) -> Image.Image:
    w, h = image.size
    # get the largest dimension so we can pad to a square
    px = max(image.size)
    # pad to square with white background
    canvas = Image.new("RGB", (px, px), (255, 255, 255))
    canvas.paste(image, ((px - w) // 2, (px - h) // 2))
    return canvas


import numpy as np

#==================== 获取标签分组列表 =====================
@dataclass
class LabelData:
    names: list[str]
    rating: list[np.int64]
    general: list[np.int64]
    character: list[np.int64]

#获取标签列表
from typing import Optional
from pathlib import Path
import pandas as pd
from huggingface_hub import hf_hub_download
from huggingface_hub.utils import HfHubHTTPError
def load_labels_hf(
    repo_id: str,
    revision: Optional[str] = None,
    token: Optional[str] = None,
) -> LabelData:
    try:
        csv_path = hf_hub_download(
            repo_id=repo_id, filename="selected_tags.csv", revision=revision, token=token
        )
        csv_path = Path(csv_path).resolve()
    except HfHubHTTPError as e:
        raise FileNotFoundError(f"selected_tags.csv failed to download from {repo_id}") from e

    df: pd.DataFrame = pd.read_csv(csv_path, usecols=["name", "category"])
    tag_data = LabelData(
        names=df["name"].tolist(),
        rating=list(np.where(df["category"] == 9)[0]),
        general=list(np.where(df["category"] == 0)[0]),
        character=list(np.where(df["category"] == 4)[0]),
    )

    return tag_data


from typing import Tuple
from torch import Tensor

 
#===================      模型加载和复用  =================================

_model_cache = {}
_labels_cache = {}
_transform_cache = {}

def load_model_and_labels(model_name: str):
    """加载并缓存模型、标签和transform"""
    if model_name in _model_cache:
        return _model_cache[model_name], _labels_cache[model_name], _transform_cache[model_name]

    repo_id = MODEL_REPO_MAP.get(model_name)
    if not repo_id:
        raise ValueError(f"未知模型: {model_name}")

    print(f"[INIT] 加载模型 {model_name} ({repo_id}) ...")
    model: nn.Module = timm.create_model("hf-hub:" + repo_id).eval()
    state_dict = timm.models.load_state_dict_from_hf(repo_id)
    model.load_state_dict(state_dict)

    labels = load_labels_hf(repo_id=repo_id)
    transform = create_transform(**resolve_data_config(model.pretrained_cfg, model=model))

    # 缓存
    _model_cache[model_name] = model
    _labels_cache[model_name] = labels
    _transform_cache[model_name] = transform

    return model, labels, transform



def get_tags(
    probs: Tensor,
    labels: LabelData,
    gen_threshold: float,
    char_threshold: float,
):
    # Convert indices+probs to labels
    probs = list(zip(labels.names, probs.numpy()))

    # First 4 labels are actually ratings
    rating_labels = dict([probs[i] for i in labels.rating])

    # General labels, pick any where prediction confidence > threshold
    gen_labels = [probs[i] for i in labels.general]
    gen_labels = dict([x for x in gen_labels if x[1] > gen_threshold])
    gen_labels = dict(sorted(gen_labels.items(), key=lambda item: item[1], reverse=True))

    # Character labels, pick any where prediction confidence > threshold
    char_labels = [probs[i] for i in labels.character]
    char_labels = dict([x for x in char_labels if x[1] > char_threshold])
    char_labels = dict(sorted(char_labels.items(), key=lambda item: item[1], reverse=True))

    # Combine general and character labels, sort by confidence
    combined_names = [x for x in gen_labels]
    combined_names.extend([x for x in char_labels])

    # Convert to a string suitable for use as a training caption
    caption = ", ".join(combined_names)
    taglist = caption.replace("_", " ").replace("(", "\(").replace(")", "\)")

    return caption, taglist, rating_labels, char_labels, gen_labels


@dataclass
class SDXL_Options:
    image_base64: str = ""
    model: str = "vit"
    gen_threshold: float = 0.35
    char_threshold: float = 0.75

import torch
import torch.nn as nn
import torch.nn.functional as F
import timm
from simple_parsing import field, parse_known_args
from timm.data import create_transform, resolve_data_config
from torch import Tensor
from torch.cuda import is_available as torch_cuda_is_available
from dataclasses import dataclass, field
torch_device = torch.device("cuda" if torch_cuda_is_available() else "cpu")

@dataclass
class TaggingResult:
    caption:str  # 自动生成的标签文本
    flag : bool=True  # 是否成功获取标签
    model : str= "WD"  # 模型名称






from io import BytesIO

#=====================   主方法  ================================#
def auto_tagger(opts: SDXL_Options):
    try:
        if not opts.image_base64:
            raise ValueError("image_base64 is empty")
        image_bytes = base64.b64decode(opts.image_base64)
        img_input=Image.open(BytesIO(image_bytes))
        model,labels,transform = load_model_and_labels(opts.model)
        print("loaded model and labels")
        print("Loading image and preprocessing...")
        # ensure image is RGB
        img_input = pil_ensure_rgb(img_input)
        # pad to square with white background
        img_input = pil_pad_square(img_input)
        # run the model's input transform to convert to tensor and rescale
        inputs = transform(img_input).unsqueeze(0)
        # NCHW image RGB to BGR
        inputs = inputs[:, [2, 1, 0]]
        print("Running inference...")
        with torch.inference_mode():
            # move model to GPU, if available
            if torch_device.type != "cpu":
                model = model.to(torch_device)
                inputs = inputs.to(torch_device)
            # run the model
            outputs : Tensor = model.forward(inputs)
            # apply the final activation function (timm doesn't support doing this internally)
            outputs = F.sigmoid(outputs)
            # move inputs, outputs, and model back to to cpu if we were on GPU
            if torch_device.type != "cpu":
                inputs = inputs.to("cpu")
                outputs = outputs.to("cpu")
                model = model.to("cpu")
            print("Processing results...")
            caption, taglist, ratings, character, general = get_tags(
                probs=outputs.squeeze(0),
                labels=labels,
                gen_threshold=opts.gen_threshold,
                char_threshold=opts.char_threshold,
                )
            return TaggingResult(caption=caption, flag=True)
    except Exception as e:
        print(f"Error in use WD model to tagging image: {e}")
        return TaggingResult(caption="", flag=False)


__all__=[
    "auto_tagger"
]
""" 

使用方法auto_tagger(opts: ScriptOptions)

opts: ScriptOptions 类，包含以下参数：

- image_file:str = ""  # 图片路径
- model: str = field(default="vit")  # 模型名称，可选值：vit、swinv2、convnext
- gen_threshold: float = field(default=0.35)  # 通用标签阈值
- char_threshold: float = field(default=0.75)  # 角色标签阈值

返回值：
 caption:str  # 自动生成的标签文本




"""



if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--image_file", type=str, help="Path to image file",default="test.jpg")
    parser.add_argument("--model", type=str, default="vit", choices=["vit", "swinv2", "convnext"], help="Model to use")
    parser.add_argument("--gen_threshold", type=float, default=0.35, help="General label threshold")
    parser.add_argument("--char_threshold", type=float, default=0.75, help="Character label threshold")
    args = parser.parse_args()
    print(auto_tagger(SDXL_Options(**vars(args))))


