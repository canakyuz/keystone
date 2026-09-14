#!/usr/bin/env python3
"""Cuts the console's engraving plates from public-domain scans on Wikimedia Commons.

Each scan is fetched by name, checked against the SHA-1 Commons reports for it, reduced to
one bit, stripped of lone specks, trimmed to the ink, and written as an SVG of merged pixel
rectangles. The console draws each SVG as a mask over currentColor, so one file holds on
paper and in the dark theme. lib/plates.ts is written alongside with the sizes, alt text and
credits, so a page never hand-copies them.

Plates are thresholded: Floyd-Steinberg breaks a thin engraved line into a dotted one, and on
the small Diderot scan it turned the hatching to noise. "dither" stays for a tonal source.

Run from console/:  python3 scripts/plates.py   (needs Pillow)
Time: O(W x H) per plate. Space: O(W x H) for the working bitmap.
"""

import hashlib
import io
import json
import urllib.parse
import urllib.request
from pathlib import Path

from PIL import Image, ImageOps

ROOT = Path(__file__).resolve().parent.parent
SVG_DIR = ROOT / "public" / "plates"
MANIFEST = ROOT / "lib" / "plates.ts"
USER_AGENT = "keystone-console-plates/1.0 (https://github.com/canakyuz/keystone)"

PLATES = [
    {
        "key": "vault",
        "file": "Construction.voute.romaine.png",
        "sha1": "8992ec1d3224e140b661332d3d0000270cf7a4c1",
        "width": 460,
        "mode": "threshold",
        "cut": 150,
        "alt": "Engraving of a Roman barrel vault laid stone by stone over its timber centering, a mason standing below",
        "credit": "Viollet-le-Duc, Dictionnaire raisonné de l'architecture, 1856",
    },
    {
        "key": "pier",
        "file": "Tas.de.charge.2.png",
        "sha1": "3ecf2cb52518d7eaa5d04dbf9baf29cd7ae7e9d6",
        "width": 400,
        "mode": "threshold",
        "cut": 185,
        "alt": "Engraving of two arches springing from a single pier",
        "credit": "Viollet-le-Duc, Dictionnaire raisonné de l'architecture, 1856",
    },
    {
        "key": "masons",
        "file": "Engraving from Diderot, Encyclopédie, v. 1, pl. 194, Architecture Maconnerie. Masonry arch. LCCN2006677828.jpg",
        "sha1": "cb97e6033d88278ba88634e87530c24d168db021",
        # Kept at the scan's own width: this scan is small, and every reduction merged the
        # hatching into grey that the threshold then turned to blots.
        "width": 604,
        "mode": "threshold",
        "cut": 110,
        # The upper scene only; the plate's lower half is figure diagrams of brick bonds.
        "crop": (18, 34, 622, 322),
        "alt": "Engraving of masons cutting and setting stone in a yard before a brick arch",
        "credit": "Diderot and d'Alembert, Encyclopédie, Maçonnerie plate I, 1762",
    },
]


def fetch(file_name: str, sha1: str) -> Image.Image:
    url = "https://commons.wikimedia.org/wiki/Special:FilePath/" + urllib.parse.quote(file_name.replace(" ", "_"))
    with urllib.request.urlopen(urllib.request.Request(url, headers={"User-Agent": USER_AGENT})) as response:
        data = response.read()
    digest = hashlib.sha1(data).hexdigest()
    if digest != sha1:
        raise SystemExit(f"{file_name}: expected sha1 {sha1}, got {digest}; the file on Commons has changed")
    return Image.open(io.BytesIO(data))


def one_bit(image: Image.Image, plate: dict) -> list[list[bool]]:
    grey = image.convert("L")
    if "crop" in plate:
        grey = grey.crop(plate["crop"])
    grey = ImageOps.autocontrast(grey, cutoff=1)
    height = round(grey.height * plate["width"] / grey.width)
    grey = grey.resize((plate["width"], height), Image.LANCZOS)
    if plate["mode"] == "dither":
        bits = grey.convert("1", dither=Image.FLOYDSTEINBERG)
    else:
        bits = grey.point(lambda v: 255 if v > plate["cut"] else 0).convert("1", dither=Image.NONE)
    px = bits.load()
    return [[px[x, y] == 0 for x in range(bits.width)] for y in range(bits.height)]


def despeckle(ink: list[list[bool]]) -> list[list[bool]]:
    """Drops ink pixels with no inked neighbour: scan noise, and path data spent on nothing."""
    h, w = len(ink), len(ink[0])

    def has_neighbour(x: int, y: int) -> bool:
        return any(
            ink[ny][nx]
            for ny in range(max(0, y - 1), min(h, y + 2))
            for nx in range(max(0, x - 1), min(w, x + 2))
            if (nx, ny) != (x, y)
        )

    return [[ink[y][x] and has_neighbour(x, y) for x in range(w)] for y in range(h)]


def trim(ink: list[list[bool]]) -> list[list[bool]]:
    rows = [y for y, row in enumerate(ink) if any(row)]
    cols = [x for x in range(len(ink[0])) if any(row[x] for row in ink)]
    return [row[cols[0] : cols[-1] + 1] for row in ink[rows[0] : rows[-1] + 1]]


def runs(row: list[bool]) -> list[tuple[int, int]]:
    spans, start = [], None
    for x, on in enumerate(row + [False]):
        if on and start is None:
            start = x
        elif not on and start is not None:
            spans.append((start, x))
            start = None
    return spans


def rectangles(ink: list[list[bool]]) -> list[tuple[int, int, int, int]]:
    """Row runs, merged downward while the next row repeats the same run exactly.

    Engraved hatching repeats run for run from one row to the next, so this roughly halves
    the path against one rectangle per run, in a single pass over the bitmap.
    """
    open_runs: dict[tuple[int, int], int] = {}
    out = []
    for y, row in enumerate(ink + [[False] * len(ink[0])]):
        current = set(runs(row))
        for span, top in list(open_runs.items()):
            if span not in current:
                out.append((span[0], top, span[1] - span[0], y - top))
                del open_runs[span]
        for span in current:
            open_runs.setdefault(span, y)
    return out


def svg(ink: list[list[bool]]) -> str:
    d = "".join(f"M{x} {y}h{w}v{h}h-{w}Z" for x, y, w, h in rectangles(ink))
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {len(ink[0])} {len(ink)}" '
        f'shape-rendering="crispEdges"><path d="{d}"/></svg>\n'
    )


def main() -> None:
    SVG_DIR.mkdir(parents=True, exist_ok=True)
    entries = {}
    for plate in PLATES:
        ink = trim(despeckle(one_bit(fetch(plate["file"], plate["sha1"]), plate)))
        body = svg(ink)
        (SVG_DIR / f"{plate['key']}.svg").write_text(body)
        entries[plate["key"]] = {
            "src": f"/plates/{plate['key']}.svg",
            "width": len(ink[0]),
            "height": len(ink),
            "alt": plate["alt"],
            "credit": plate["credit"],
            "source": "https://commons.wikimedia.org/wiki/File:" + urllib.parse.quote(plate["file"].replace(" ", "_")),
        }
        print(f"{plate['key']}: {len(ink[0])}x{len(ink)}, {len(body) // 1024} KB")

    lines = ["// Written by scripts/plates.py. Edit the script, not this file.", "", "export const PLATES = {"]
    for key, entry in entries.items():
        fields = ", ".join(f"{name}: {json.dumps(value, ensure_ascii=False)}" for name, value in entry.items())
        lines.append(f"  {key}: {{ {fields} }},")
    lines += ["} as const;", "", "export type PlateName = keyof typeof PLATES;", ""]
    MANIFEST.write_text("\n".join(lines))


if __name__ == "__main__":
    main()
