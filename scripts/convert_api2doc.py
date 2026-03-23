#!/usr/bin/env python3
"""
Convert HTML documentation file `docs/api2doc` to Markdown using markdownify.

Usage:
  python3 scripts/convert_api2doc.py \
    --input ../docs/api2doc \
    --output ../docs/RegAPI2.md

The script uses the `markdownify` package. Install with:
  python3 -m pip install markdownify
"""
import argparse
from pathlib import Path
import sys

def main():
    parser = argparse.ArgumentParser(description='Convert HTML doc to Markdown')
    parser.add_argument('--input', '-i', required=True, help='Path to input HTML file')
    parser.add_argument('--output', '-o', required=True, help='Path to output Markdown file')
    args = parser.parse_args()

    try:
        from markdownify import markdownify
    except Exception as e:
        print('Missing dependency: markdownify. Install with: python3 -m pip install markdownify', file=sys.stderr)
        raise

    in_path = Path(args.input).expanduser().resolve()
    out_path = Path(args.output).expanduser().resolve()

    if not in_path.exists():
        print(f'Input file not found: {in_path}', file=sys.stderr)
        sys.exit(2)

    html = in_path.read_text(encoding='utf-8')

    # Convert to markdown. Use heading style ATX and preserve code blocks.
    md = markdownify(html, heading_style='ATX')

    # Basic post-processing: normalize multiple blank lines
    import re
    md = re.sub(r"\n{3,}", "\n\n", md)

    out_path.write_text(md, encoding='utf-8')
    print(f'Converted {in_path} -> {out_path}')

if __name__ == '__main__':
    main()

