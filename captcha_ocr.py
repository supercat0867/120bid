import json
import sys
import ddddocr
from PIL import Image
import os
import contextlib

# 兼容 Pillow 版本差异
if not hasattr(Image, 'ANTIALIAS'):
    setattr(Image, 'ANTIALIAS', Image.LANCZOS)


def recognize_captcha(image_path: str) -> str:
    """识别验证码并返回识别结果"""
    # 屏蔽 ddddocr 的打印信息
    with open(os.devnull, 'w') as fnull, contextlib.redirect_stdout(fnull), contextlib.redirect_stderr(fnull):
        ocr = ddddocr.DdddOcr()  # 默认模型
    with open(image_path, "rb") as f:
        img_bytes = f.read()
    return ocr.classification(img_bytes)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "missing image path"}))
        sys.exit(1)

    image_path = sys.argv[1]
    try:
        result = recognize_captcha(image_path)
        print(json.dumps({"text": result}))
    except Exception as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)