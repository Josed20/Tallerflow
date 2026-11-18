"""Render Android assets from the official SVG. Requires Inkscape on PATH (or INKSCAPE)."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
INKSCAPE = os.environ.get("INKSCAPE") or shutil.which("inkscape")
if not INKSCAPE:
    raise SystemExit("Set INKSCAPE to the Inkscape executable.")
symbol = (ROOT / "frontend/public/logo-symbol-on-dark.svg").read_text(encoding="utf-8")
symbol = symbol[symbol.index("<g "):symbol.rindex("</svg>")]
RES = ROOT / "frontend/android/app/src/main/res"


def render(target, size, foreground=False, splash=False):
    target.parent.mkdir(parents=True, exist_ok=True)
    # The foreground stays inside Android's circular safe zone. Let the OS mask the background.
    width = 146 if foreground else (150 if splash else 180)
    height = width * 106.51 / 146.11
    x, y = (288-width)/2, (288-height)/2
    bg = "" if foreground else '<rect width="288" height="288" fill="#062550"/>'
    svg = f'<svg xmlns="http://www.w3.org/2000/svg" width="288" height="288" viewBox="0 0 288 288">{bg}<g transform="translate({x} {y}) scale({width/146.11})">{symbol}</g></svg>'
    with tempfile.TemporaryDirectory() as temp:
        source = Path(temp) / "brand.svg"
        source.write_text(svg, encoding="utf-8")
        subprocess.run([INKSCAPE, str(source), "--export-type=png", f"--export-filename={target}", f"--export-width={size}", f"--export-height={size}"], check=True, capture_output=True)


for density, scale in [("mdpi", 1), ("hdpi", 1.5), ("xhdpi", 2), ("xxhdpi", 3), ("xxxhdpi", 4)]:
    for name in ["ic_launcher", "ic_launcher_round"]:
        render(RES / f"mipmap-{density}/{name}.png", round(48 * scale))
    render(RES / f"mipmap-{density}/ic_launcher_foreground.png", round(108 * scale), foreground=True)
render(ROOT / "frontend/resources/icon.png", 1024)
render(ROOT / "frontend/resources/icon-foreground.png", 1024, foreground=True)
render(ROOT / "frontend/resources/splash.png", 2732, splash=True)
# Use a density-independent centered bitmap instead of stretched landscape/portrait bitmaps.
render(RES / "drawable-nodpi/brand_splash.png", 384, foreground=True)
(RES / "drawable/splash.xml").write_text('''<?xml version="1.0" encoding="utf-8"?>
<layer-list xmlns:android="http://schemas.android.com/apk/res/android">
    <item android:drawable="@color/ic_launcher_background" />
    <item><bitmap android:src="@drawable/brand_splash" android:gravity="center" /></item>
</layer-list>
''', encoding="utf-8")
for old in RES.glob("drawable*/splash.png"):
    old.unlink()
background = ROOT / "frontend/resources/icon-background.png"
with tempfile.TemporaryDirectory() as temp:
    source = Path(temp) / "background.svg"
    source.write_text('<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024"><rect width="1024" height="1024" fill="#062550"/></svg>')
    subprocess.run([INKSCAPE, str(source), "--export-type=png", f"--export-filename={background}"], check=True, capture_output=True)
print("Android assets rendered from the official TallerFlow logo.")
