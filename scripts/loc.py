#!/usr/bin/env python3
"""
Count lines of code for selected file types:
- .py, .yaml, .yml, .sh, .json
- Dockerfile and Dockerfile.*
- Justfile

Usage:
  python -m scripts.loc [path]
  python scripts/loc.py [path]
"""

from __future__ import annotations

import sys
from collections.abc import Iterable
from pathlib import Path

EXCLUDE_DIRS = {
    ".git",
    ".hg",
    ".svn",
    ".venv",
    "venv",
    "__pycache__",
    ".mypy_cache",
    ".pytest_cache",
    ".ruff_cache",
    "node_modules",
    "dist",
    "build",
    "test-data",
}


# Map of label -> matcher function
def is_dockerfile(p: Path) -> bool:
    name = p.name
    return name == "Dockerfile" or name.startswith("Dockerfile.")


def is_justfile(p: Path) -> bool:
    return p.name == "Justfile"


MATCHERS: dict[str, callable[[Path], bool]] = {
    "go(.go)": lambda p: p.suffix == ".go",
    "python(.py)": lambda p: p.suffix == ".py",
    "yaml(.yaml)": lambda p: p.suffix == ".yaml",
    "yaml(.yml)": lambda p: p.suffix == ".yml",
    "shell(.sh)": lambda p: p.suffix == ".sh",
    "json(.json)": lambda p: p.suffix == ".json",
    "docker(Dockerfile)": is_dockerfile,
    "just(Justfile)": is_justfile,
}


def iter_files(root: Path) -> Iterable[Path]:
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        # Skip files inside excluded directories
        parts = set(path.parts)
        if parts & EXCLUDE_DIRS:
            # Fast path: if any excluded directory is in the path parts
            # This works because directory names become parts
            continue
        yield path


def count_lines(path: Path) -> int:
    try:
        with path.open("r", encoding="utf-8", errors="replace") as f:
            # Sum lines efficiently without loading entire file
            return sum(1 for _ in f)
    except OSError:
        return 0


def main(argv: list[str]) -> int:
    root = Path(argv[1]) if len(argv) > 1 else Path(".")
    root = root.resolve()

    print(f"Counting lines under: {root}")
    print("Excluding: " + ", ".join(sorted(EXCLUDE_DIRS)))
    print()

    totals: dict[str, tuple[int, int]] = {}  # label -> (files, lines)

    files = list(iter_files(root))
    for label, matcher in MATCHERS.items():
        sel = [p for p in files if matcher(p)]
        line_sum = 0
        for p in sel:
            line_sum += count_lines(p)
        totals[label] = (len(sel), line_sum)
        print(f"{label:20s} files:{len(sel):3d}      lines:{line_sum:5d}")

    grand_total = sum(lines for _, lines in totals.values())
    print()
    print(f"{'Grand total lines:':20s} {grand_total:20d}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
