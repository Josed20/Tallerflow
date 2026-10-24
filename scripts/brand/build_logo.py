"""Builds the TallerFlow woven-diamond logo SVGs from one geometric definition.

Usage: python scripts/brand/build_logo.py <out-dir> <PlusJakartaSans[wght].ttf>
Needs fonttools. The font (OFL) comes from github.com/google/fonts, ofl/plusjakartasans;
it is only used to outline the wordmark, never shipped. Copy the SVGs you need into
frontend/public/. PNG icons are rasterized from these SVGs (see DESIGN.md, Logo).

The mark, rotated 45 degrees, is a 4x4 woven grid. Cells are (col, row);
each piece lists the cells it spans and which of its corners are rounded.
"""
import sys
from pathlib import Path

CELL = 24
GAP = 4
R = 8  # outer corner radius
STEP = CELL + GAP
SIZE = 4 * CELL + 3 * GAP  # 108

NAVY = "#0A346F"
TEAL = "#00BFB2"

# (color, [(col0, row0, col1, row1) rectangles], rounded corners per rectangle "tl tr br bl")
PIECES = [
    ("teal", [(0, 0, 1, 0)], ["tl tr"]),                       # top bar
    ("navy", [(2, 0, 2, 1)], ["tl tr br"]),                    # right-top post
    ("navy", [(0, 1, 0, 2)], ["tl bl"]),                       # left post
    ("teal", [(1, 1, 1, 1)], [""]),                            # centre
    ("teal", [(1, 2, 1, 2)], [""]),                            # centre-bottom
    ("navy", [(2, 2, 2, 2)], ["br"]),                          # centre-right
    ("teal", "L", None),                                       # outer L: col 3 rows 1-3 + row 3 cols 1-3
]


def box(c0, r0, c1, r1):
    x0, y0 = c0 * STEP, r0 * STEP
    x1, y1 = c1 * STEP + CELL, r1 * STEP + CELL
    return x0, y0, x1, y1


def rounded_rect(x0, y0, x1, y1, corners):
    tl, tr, br, bl = (R if k in corners.split() else 0 for k in ("tl", "tr", "br", "bl"))
    return (
        f"M{x0 + tl} {y0}H{x1 - tr}" + (f"A{tr} {tr} 0 0 1 {x1} {y0 + tr}" if tr else "")
        + f"V{y1 - br}" + (f"A{br} {br} 0 0 1 {x1 - br} {y1}" if br else "")
        + f"H{x0 + bl}" + (f"A{bl} {bl} 0 0 1 {x0} {y1 - bl}" if bl else "")
        + f"V{y0 + tl}" + (f"A{tl} {tl} 0 0 1 {x0 + tl} {y0}" if tl else "") + "Z"
    )


def l_shape():
    # Vertical arm: col 3, rows 1-3. Horizontal arm: row 3, cols 1-3.
    vx0, vy0, vx1, _ = box(3, 1, 3, 1)
    hx0, hy0, _, hy1 = box(1, 3, 1, 3)
    return (
        f"M{vx0} {vy0}H{vx1 - R}A{R} {R} 0 0 1 {vx1} {vy0 + R}"
        f"V{hy1 - R}A{R} {R} 0 0 1 {vx1 - R} {hy1}"
        f"H{hx0 + R}A{R} {R} 0 0 1 {hx0} {hy1 - R}"
        f"V{hy0}H{vx0}Z"
    )


def mark_paths(navy, teal):
    out = []
    for color, rects, corners in PIECES:
        fill = navy if color == "navy" else teal
        d = l_shape() if rects == "L" else "".join(rounded_rect(*box(*r), c) for r, c in zip(rects, corners))
        out.append(f'<path fill="{fill}" d="{d}"/>')
    return "\n    ".join(out)


# The grid is drawn axis-aligned, then turned 45 degrees counter-clockwise into the diamond.
HALF = SIZE / 2
DIAG = SIZE * 2 ** 0.5


def _extent():
    """Half width and half height of the rotated silhouette. The two empty cells clip
    the top and bottom tips, so the diamond is wider than tall."""
    import math

    pts = []
    for _, rects, corners in PIECES:
        boxes = [(box(3, 1, 3, 3), "tr br"), (box(1, 3, 3, 3), "bl br")] if rects == "L" else [
            (box(*r), c) for r, c in zip(rects, corners)]
        for (x0, y0, x1, y1), cs in boxes:
            for name, (cx, cy, a0) in {"tl": (x0 + R, y0 + R, 180), "tr": (x1 - R, y0 + R, 270),
                                      "br": (x1 - R, y1 - R, 0), "bl": (x0 + R, y1 - R, 90)}.items():
                if name in cs.split():
                    pts += [(cx + R * math.cos(math.radians(a0 + k * 9)), cy + R * math.sin(math.radians(a0 + k * 9))) for k in range(11)]
                else:
                    pts.append({"tl": (x0, y0), "tr": (x1, y0), "br": (x1, y1), "bl": (x0, y1)}[name])
    c = math.sqrt(0.5)
    xs = [((x - HALF) + (y - HALF)) * c for x, y in pts]   # rotate(-45): x' = (x + y)/sqrt2
    ys = [(-(x - HALF) + (y - HALF)) * c for x, y in pts]  #              y' = (y - x)/sqrt2
    return max(map(abs, xs)), max(map(abs, ys))


HW, HH = _extent()
MARK_W, MARK_H = 2 * HW, 2 * HH


def mark_group(navy, teal, x=0, y=0, scale=1.0):
    t = f"translate({x + HW * scale:.3f} {y + HH * scale:.3f}) scale({scale:.4f}) rotate(-45) translate({-HALF} {-HALF})"
    return f'<g transform="{t}">\n    {mark_paths(navy, teal)}\n  </g>'


def symbol_svg(navy=NAVY, teal=TEAL, title="TallerFlow"):
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {MARK_W:.2f} {MARK_H:.2f}" role="img" aria-label="{title}">\n'
        f"  {mark_group(navy, teal)}\n</svg>\n"
    )


def app_icon_svg(size=512, inset=0.16, radius=0.22, background=NAVY, navy="#FFFFFF", teal=TEAL):
    """Square icon: the mark on a rounded navy tile (favicon, PWA, Android)."""
    scale = size * (1 - 2 * inset) / MARK_W
    x, y = (size - MARK_W * scale) / 2, (size - MARK_H * scale) / 2
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {size} {size}" role="img" aria-label="TallerFlow">\n'
        f'  <rect width="{size}" height="{size}" rx="{size * radius:.1f}" fill="{background}"/>\n'
        f"  {mark_group(navy, teal, x, y, scale)}\n</svg>\n"
    )


def wordmark_paths(font_path, words, cap_height, tracking=-0.02):
    """Outlines each (text, color) run with Plus Jakarta Sans ExtraBold. Returns (svg, width)."""
    from fontTools.pens.svgPathPen import SVGPathPen
    from fontTools.pens.transformPen import TransformPen
    from fontTools.ttLib import TTFont
    from fontTools.varLib.instancer import instantiateVariableFont

    font = instantiateVariableFont(TTFont(font_path), {"wght": 800})
    upm = font["head"].unitsPerEm
    cap = font["OS/2"].sCapHeight
    scale = cap_height / cap
    cmap, glyphs, hmtx = font.getBestCmap(), font.getGlyphSet(), font["hmtx"]
    x, parts = 0.0, []
    for text, color in words:
        pen = SVGPathPen(glyphs)
        for ch in text:
            name = cmap[ord(ch)]
            # Flip y: font units grow upward, SVG downward; baseline sits at y = cap_height.
            glyphs[name].draw(TransformPen(pen, (scale, 0, 0, -scale, x, cap_height)))
            x += hmtx[name][0] * scale + tracking * upm * scale
        parts.append(f'<path fill="{color}" d="{pen.getCommands()}"/>')
    return "\n  ".join(parts), x - tracking * upm * scale


def horizontal_svg(font_path, navy, teal, word_navy, word_teal, height=100):
    """Mark plus wordmark; cap height is half the mark, centred on it, like the PNG."""
    scale = height / MARK_H
    cap = height * 0.5
    words, width = wordmark_paths(font_path, [("Taller", word_navy), ("Flow", word_teal)], cap)
    gap = height * 0.07
    tx, ty = MARK_W * scale + gap, (height - cap) / 2
    total = tx + width
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {total:.2f} {height}" role="img" aria-label="TallerFlow">\n'
        f"  {mark_group(navy, teal, scale=scale)}\n"
        f'  <g transform="translate({tx:.2f} {ty:.2f})">\n  {words}\n  </g>\n</svg>\n'
    )


if __name__ == "__main__":
    out = Path(sys.argv[1])
    font_path = sys.argv[2]
    out.mkdir(parents=True, exist_ok=True)
    white = "#FFFFFF"
    files = {
        "logo-symbol.svg": symbol_svg(),
        "logo-symbol-on-dark.svg": symbol_svg(navy=white),
        "logo-monochrome.svg": symbol_svg(navy="currentColor", teal="currentColor"),
        "logo-horizontal.svg": horizontal_svg(font_path, NAVY, TEAL, NAVY, TEAL),
        "logo-horizontal-on-dark.svg": horizontal_svg(font_path, white, TEAL, white, TEAL),
        "favicon.svg": app_icon_svg(),
        "app-icon.svg": app_icon_svg(),
        # Full-bleed square: Android/PWA maskable and iOS apply their own mask, so no radius,
        # and the mark stays inside the central safe zone.
        "app-icon-maskable.svg": app_icon_svg(radius=0, inset=0.22),
        "app-icon-square.svg": app_icon_svg(radius=0, inset=0.16),
        # Android adaptive foreground: mark only, inside the 66% safe zone.
        "icon-foreground.svg": app_icon_svg(background="none", inset=0.25).replace(' fill="none"', ' fill="none" fill-opacity="0"'),
    }
    for name, svg in files.items():
        (out / name).write_text(svg, encoding="utf-8")
    print("diamond", round(DIAG, 2), "files", list(files))
