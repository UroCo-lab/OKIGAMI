"""Rasterize the offline message using the already bundled OFL font."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

root = Path(__file__).resolve().parent.parent
font = root / 'src/web/fonts/bizudgothic-BIZUDGothic-Regular.ttf'
image = Image.new('L', (800, 300), 255)
draw = ImageDraw.Draw(image)
for text, y, size, ink in [
    ('Wi-Fi 未接続', 65, 52, 0),
    ('保存した画面は、そのまま表示できます', 150, 30, 85),
    ('書き換えるときは Wi-Fi に接続してください', 210, 28, 85),
]:
    draw.text((400, y), text, font=ImageFont.truetype(str(font), size), fill=ink, anchor='mm')
image = image.point(lambda value: min(255, ((value + 8) // 17) * 17))
image.save(root / 'src/assets/offline-label.png', optimize=True)
