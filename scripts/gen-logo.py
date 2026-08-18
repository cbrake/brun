"""Generate the [brun] logo assets in assets/.

Renders "[brun]" as embedded glyph outlines (no font dependency in the
SVG) using Fira Code Medium, in blue on black.

Usage:
    python scripts/gen-logo.py
    rsvg-convert -w 800 assets/brun-logo.svg -o assets/brun-logo.png
    rsvg-convert -w 360 -h 360 assets/brun-logo-square.svg -o assets/brun-logo-square.png

Requires fontTools and the Fira Code font (path below).
"""
from pathlib import Path

from fontTools.misc.transform import Transform
from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont

FONT = "/usr/share/fonts/TTF/FiraCodeNerdFont-Medium.ttf"
TEXT = "[brun]"
COLOR = "#4da6ff"  # blue
BG = "#000000"

ASSETS = Path(__file__).resolve().parent.parent / "assets"

font = TTFont(FONT)
cmap = font.getBestCmap()
glyphs = font.getGlyphSet()

# Build one combined path, y-flipped (font coords -> SVG coords), advancing x.
pen = SVGPathPen(glyphs)
bpen = BoundsPen(glyphs)
x = 0
for ch in TEXT:
    g = glyphs[cmap[ord(ch)]]
    t = Transform(1, 0, 0, -1, x, 0)
    g.draw(TransformPen(pen, t))
    g.draw(TransformPen(bpen, t))
    x += g.width
path = pen.getCommands()
xmin, ymin, xmax, ymax = bpen.bounds
w, h = xmax - xmin, ymax - ymin


def svg(width, height, scale, tx, ty):
    return f"""<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {width} {height}" width="{width}" height="{height}">
  <rect width="{width}" height="{height}" fill="{BG}"/>
  <g transform="translate({tx:.2f}, {ty:.2f}) scale({scale:.6f})">
    <path d="{path}" fill="{COLOR}"/>
  </g>
</svg>
"""


# Wide wordmark, normalized to 100 px tall.
pad = 0.30 * h
s = 100.0 / (h + 2 * pad)
W = round((w + 2 * pad) * s, 2)
(ASSETS / "brun-logo.svg").write_text(
    svg(W, 100, s, (pad - xmin) * s, (pad - ymin) * s)
)
print(f"wrote assets/brun-logo.svg {W} x 100")

# Square logo: 360x360, text ~78% of width, centered.
SQ = 360
s2 = (SQ * 0.78) / w
tx = (SQ - w * s2) / 2 - xmin * s2
ty = (SQ - h * s2) / 2 - ymin * s2
(ASSETS / "brun-logo-square.svg").write_text(svg(SQ, SQ, s2, tx, ty))
print("wrote assets/brun-logo-square.svg 360x360")
