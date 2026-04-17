#!/usr/bin/env bash
# Render the Jotter otter logo to braille ASCII art.
#
# Produces two outputs from the same source PNG:
#   - assets/jotter-the-otter.txt (100x50) — canonical full-size art
#   - cmd/banner.txt (50x25) — scaled-down version embedded in the CLI
#
# Pipeline: chafa renders the PNG as braille with no colour, then a
# declutter pass clears isolated specks (cells with <=2 dots whose 8
# neighbours collectively have <=3 dots).
#
# Source image should be dark background + light subject — chafa's
# braille mode treats bright pixels as "on", so that maps the otter
# silhouette to dots and leaves negative space empty.
#
# Usage: scripts/render-logo.sh [source.png]
# Requires: chafa, python3.

set -euo pipefail

SRC="${1:-assets/jotter-the-otter-ascii-template.png}"

cd "$(git rev-parse --show-toplevel)"

render() {
  local size="$1" out="$2" trim="${3:-false}"
  local tmp pre
  tmp=$(mktemp)
  pre=$(mktemp).png
  # Dilate bright pixels by 1 — thickens the thin white silhouette outline
  # so it survives braille-resolution sampling without gaps.
  magick "$SRC" -morphology Dilate Disk:1 "$pre"
  chafa \
    --format=symbols \
    --symbols=braille \
    --size="$size" \
    --fg-only \
    --colors=none \
    "$pre" \
    | python3 scripts/declutter.py /dev/stdin \
    > "$tmp"
  rm -f "$pre"
  if [[ "$trim" == true ]]; then
    # Strip leading and trailing blank (all-U+2800) lines.
    awk '/^⠀*$/{if(seen)buf=buf $0 "\n"; next} {printf "%s%s\n",buf,$0; buf=""; seen=1}' "$tmp" > "$out"
  else
    cp "$tmp" "$out"
  fi
  rm -f "$tmp"
  echo "wrote $out ($size)"
}

render 100x50 assets/jotter-the-otter.txt
render 70x35  cmd/banner.txt true

# Append a figlet "Jotter" wordmark to the embedded banner — the book-cover
# text in the braille render is illegible at small sizes, so we draw it
# separately in ASCII for guaranteed legibility.
figlet -f big -w 70 -c "Jotter" >> cmd/banner.txt
echo "appended figlet wordmark to cmd/banner.txt"
