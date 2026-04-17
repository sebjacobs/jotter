#!/usr/bin/env python3
"""Remove isolated sparse braille cells from ASCII art.

Heuristic: a cell is cleared if it has <=self_max dots AND
the 8 neighbors collectively have <=neighbor_max dots.
"""
import sys

BLANK = "\u2800"

def dots(ch: str) -> int:
    cp = ord(ch)
    if not (0x2800 <= cp <= 0x28FF):
        return 0
    return bin(cp - 0x2800).count("1")

def clean(lines, self_max=2, neighbor_max=3):
    grid = [list(l.rstrip("\n")) for l in lines]
    h = len(grid)
    w = max(len(r) for r in grid)
    for r in grid:
        while len(r) < w:
            r.append(BLANK)
    out = [row[:] for row in grid]
    cleared = 0
    for y in range(h):
        for x in range(w):
            d = dots(grid[y][x])
            if d == 0 or d > self_max:
                continue
            neigh = 0
            for dy in (-1, 0, 1):
                for dx in (-1, 0, 1):
                    if dy == 0 and dx == 0:
                        continue
                    ny, nx = y + dy, x + dx
                    if 0 <= ny < h and 0 <= nx < w:
                        neigh += dots(grid[ny][nx])
            if neigh <= neighbor_max:
                out[y][x] = BLANK
                cleared += 1
    return ["".join(r) for r in out], cleared

if __name__ == "__main__":
    path = sys.argv[1]
    self_max = int(sys.argv[2]) if len(sys.argv) > 2 else 2
    neighbor_max = int(sys.argv[3]) if len(sys.argv) > 3 else 3
    with open(path) as f:
        lines = f.readlines()
    out, n = clean(lines, self_max, neighbor_max)
    sys.stderr.write(f"cleared {n} cells\n")
    sys.stdout.write("\n".join(out) + "\n")
