#!/usr/bin/env python3
"""Convert the supplied artwork without redrawing it; build-time Pillow only."""
import argparse
from pathlib import Path
import struct
from PIL import Image, ImageOps

parser = argparse.ArgumentParser()
parser.add_argument('source', type=Path)
parser.add_argument('--output', type=Path, default=Path('cmd/nelko-print/assets'))
a = parser.parse_args()
a.output.mkdir(parents=True, exist_ok=True)
with Image.open(a.source) as original:
    original = ImageOps.exif_transpose(original).convert('RGBA')
    # Preserve the full artwork and its aspect ratio; transparent padding only.
    fitted = ImageOps.contain(original, (256, 256), Image.Resampling.LANCZOS)
    image = Image.new('RGBA', (256, 256))
    image.paste(fitted, ((256-fitted.width)//2, (256-fitted.height)//2))
    image.save(a.output/'app-icon.png')
    image.save(a.output/'app-icon.ico', format='ICO', sizes=[(16,16),(32,32),(48,48),(256,256)])
blob = (a.output/'app-icon.ico').read_bytes()
reserved, kind, count = struct.unpack_from('<HHH', blob)
assert (reserved, kind, count) == (0, 1, 4)
sizes=[]
for i in range(count):
    w,h,colors,reserved,planes,bpp,size,offset=struct.unpack_from('<BBBBHHII', blob, 6+i*16)
    sizes.append((w or 256,h or 256))
    assert offset+size <= len(blob)
assert sizes == [(16,16),(32,32),(48,48),(256,256)], sizes
print('Prepared PNG and Windows ICO:', sizes)
