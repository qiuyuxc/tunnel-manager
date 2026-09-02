#!/usr/bin/env python3
"""Split the full-coverage MiSans weights into unicode-range chunks.

Each shipped MiSans weight covers ~29.5k codepoints (~4.7 MB), so loading the
four weights whole costs every visitor ~19 MB before text settles. This script
emits, per weight:

  * one "core" file holding every character the panel's own UI renders, plus
    Latin, digits and punctuation — always fetched, ~92 KB;
  * contiguous chunks of the remaining coverage, each tagged with its own
    unicode-range, so a browser only fetches a chunk when user-supplied text
    (tunnel names, nicknames, announcements) needs a glyph from it.

Regenerate after adding UI copy is optional: unlisted characters simply come
from a chunk instead of the core file. Requires fonttools and brotli
(pip install fonttools brotli), then run: python3 scripts/subset-fonts.py
"""
from __future__ import annotations

import argparse
import concurrent.futures
import pathlib
import shutil
import subprocess
import sys
import tempfile

FRONTEND = pathlib.Path(__file__).resolve().parent.parent
SOURCE_DIR = FRONTEND / "src" / "assets" / "fonts"
OUT_DIR = FRONTEND / "public" / "fonts"

# CSS font-weight -> MiSans file stem. These are the weights styles.css uses.
WEIGHTS = {
    400: "MiSans-Regular",
    500: "MiSans-Medium",
    600: "MiSans-Semibold",
    700: "MiSans-Bold",
}

# Kept in every core file regardless of what the sources happen to contain, so
# ASCII data (domains, IPs, UUIDs) and punctuation never trigger a chunk fetch.
BASE_RANGES = [
    (0x0020, 0x007E),  # ASCII
    (0x00A0, 0x00FF),  # Latin-1 supplement
    (0x2010, 0x205E),  # general punctuation
    (0x2190, 0x21FF),  # arrows
    (0x25A0, 0x25FF),  # geometric shapes (status dots)
    (0x3000, 0x303F),  # CJK punctuation
    (0xFF00, 0xFF65),  # fullwidth forms
]

SOURCE_GLOBS = ("src/**/*.vue", "src/**/*.ts", "src/**/*.css", "index.html")


def ui_codepoints() -> set[int]:
    """Every printable character the panel's own source can render."""
    chars: set[str] = set()
    for pattern in SOURCE_GLOBS:
        for path in FRONTEND.glob(pattern):
            chars |= set(path.read_text(encoding="utf-8", errors="ignore"))
    codepoints = {ord(c) for c in chars if c.isprintable()}
    for low, high in BASE_RANGES:
        codepoints |= set(range(low, high + 1))
    return codepoints


def font_codepoints(path: pathlib.Path) -> set[int]:
    from fontTools.ttLib import TTFont

    with TTFont(path, lazy=True) as font:
        return {cp for table in font["cmap"].tables for cp in table.cmap}


def to_sfnt(source: pathlib.Path, target: pathlib.Path) -> None:
    """Inflate woff2 once so the per-chunk runs skip repeated brotli decoding."""
    from fontTools.ttLib import TTFont

    font = TTFont(source)
    font.flavor = None
    font.save(target)
    font.close()


def css_ranges(codepoints: list[int]) -> str:
    """Collapse codepoints into CSS unicode-range syntax."""
    spans: list[tuple[int, int]] = []
    start = previous = codepoints[0]
    for codepoint in codepoints[1:]:
        if codepoint == previous + 1:
            previous = codepoint
            continue
        spans.append((start, previous))
        start = previous = codepoint
    spans.append((start, previous))
    return ", ".join(
        f"U+{a:04X}" if a == b else f"U+{a:04X}-{b:04X}" for a, b in spans
    )


def write_subset(source: pathlib.Path, codepoints: list[int], target: pathlib.Path) -> int:
    with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False) as handle:
        handle.write("\n".join(f"U+{cp:04X}" for cp in codepoints))
        listing = handle.name
    try:
        subprocess.run(
            [
                "pyftsubset",
                str(source),
                f"--unicodes-file={listing}",
                "--flavor=woff2",
                f"--output-file={target}",
            ],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.PIPE,
        )
    finally:
        pathlib.Path(listing).unlink(missing_ok=True)
    return target.stat().st_size


def face_css(weight: int, filename: str, codepoints: list[int]) -> str:
    return (
        "@font-face {\n"
        '  font-family: "MiSans";\n'
        f'  src: url("/fonts/{filename}") format("woff2");\n'
        f"  font-weight: {weight};\n"
        "  font-style: normal;\n"
        "  font-display: swap;\n"
        f"  unicode-range: {css_ranges(codepoints)};\n"
        "}"
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--chunk-size", type=int, default=400, help="codepoints per chunk")
    parser.add_argument("--jobs", type=int, default=4, help="parallel pyftsubset runs")
    args = parser.parse_args()

    if not shutil.which("pyftsubset"):
        return int(print("pyftsubset not found; pip install fonttools brotli") or 1)
    missing = [s for s in WEIGHTS.values() if not (SOURCE_DIR / f"{s}.woff2").exists()]
    if missing:
        return int(print(f"missing source weights in {SOURCE_DIR}: {missing}") or 1)

    ui = ui_codepoints()
    if OUT_DIR.exists():
        shutil.rmtree(OUT_DIR)
    OUT_DIR.mkdir(parents=True)

    tasks: list[tuple[pathlib.Path, list[int], pathlib.Path]] = []
    faces: list[tuple[int, str, list[int]]] = []
    with tempfile.TemporaryDirectory() as tmp:
        for weight, stem in WEIGHTS.items():
            sfnt = pathlib.Path(tmp) / f"{stem}.ttf"
            to_sfnt(SOURCE_DIR / f"{stem}.woff2", sfnt)
            covered = font_codepoints(sfnt)
            core = sorted(covered & ui)
            rest = sorted(covered - set(core))
            parts = [("core", core)] + [
                (f"{index // args.chunk_size:03d}", rest[index:index + args.chunk_size])
                for index in range(0, len(rest), args.chunk_size)
            ]
            for label, codepoints in parts:
                filename = f"misans-{weight}-{label}.woff2"
                tasks.append((sfnt, codepoints, OUT_DIR / filename))
                faces.append((weight, filename, codepoints))
            print(f"weight {weight}: core {len(core)} cp, {len(parts) - 1} chunks")

        with concurrent.futures.ThreadPoolExecutor(args.jobs) as pool:
            sizes = list(pool.map(lambda task: write_subset(*task), tasks))

    header = "/* Generated by scripts/subset-fonts.py - do not edit by hand. */"
    css = [header] + [face_css(*face) for face in faces]
    (OUT_DIR / "fonts.css").write_text("\n".join(css) + "\n", encoding="utf-8")

    core_total = sum(size for size, face in zip(sizes, faces) if face[1].endswith("core.woff2"))
    print(f"\n{len(tasks)} files, {sum(sizes) / 1048576:.1f} MB on disk")
    print(f"core files fetched by every visit: {core_total / 1024:.0f} KB")
    print(f"average chunk: {(sum(sizes) - core_total) / max(len(tasks) - len(WEIGHTS), 1) / 1024:.0f} KB")
    return 0


if __name__ == "__main__":
    sys.exit(main())
