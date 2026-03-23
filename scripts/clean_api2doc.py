#!/usr/bin/env python3
"""
Clean converted Reg.API markdown by removing top navigation/menu clutter and large duplicated TOC.
Produces docs/RegAPI2-clean.md from docs/RegAPI2.md.

Usage:
  python3 scripts/clean_api2doc.py

"""
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
IN = ROOT / 'docs' / 'RegAPI2.md'
OUT = ROOT / 'docs' / 'RegAPI2-clean.md'

if not IN.exists():
    print('Input file not found:', IN)
    sys.exit(1)

text = IN.read_text(encoding='utf-8')

# Heuristic: find the main content start. Prefer the first occurrence of '\n# 1.' which marks the Introduction.
idx = text.find('\n# 1.')
if idx == -1:
    # Fallback: find the first top-level heading '# Документация на Рег.API 2'
    idx2 = text.find('# Документация на Рег.API 2')
    if idx2 != -1:
        # keep header but remove preceding menu; start from header
        start = idx2
    else:
        # fallback to start of file
        start = 0
else:
    start = idx + 1  # include newline before

clean = text[start:]

# Remove an immediately following large table-of-contents block if present (a table starting with '|  |')
if clean.lstrip().startswith('|  |'):
    # find end of that table (first occurrence of two newlines after the table)
    # simpler: find the first occurrence of '\n# 1.' within clean
    inner_idx = clean.find('\n# 1.')
    if inner_idx != -1:
        clean = clean[inner_idx+1:]

# Trim leading/trailing whitespace
clean = clean.lstrip('\n')

OUT.write_text(clean, encoding='utf-8')
print('Wrote cleaned file:', OUT)

