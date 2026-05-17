#!/usr/bin/env python3
"""Claude Code transcript helper for /recover-session.

Subcommands:
  list                    — list recent JSONL transcripts for the current repo
                            with timestamp + first user message preview
  extract <path> [--out]  — filter a JSONL to human/assistant text turns
                            (skips tool_use, tool_result, snapshots, etc.)
"""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path


def project_dir() -> Path:
    cwd = Path.cwd().resolve()
    # Claude Code stores transcripts under ~/.claude/projects/<cwd with / → ->
    slug = str(cwd).replace("/", "-")
    return Path.home() / ".claude" / "projects" / slug


def first_user_text(path: Path, limit: int = 200) -> str:
    with path.open() as f:
        for line in f:
            try:
                msg = json.loads(line)
            except json.JSONDecodeError:
                continue
            if msg.get("type") != "user":
                continue
            content = msg.get("message", {}).get("content", "")
            if isinstance(content, str) and content.strip():
                return content[:limit]
            if isinstance(content, list):
                for block in content:
                    if isinstance(block, dict) and block.get("type") == "text":
                        return block["text"][:limit]
    return "(no user text found)"


def mtime(path: Path) -> str:
    return subprocess.check_output(["stat", "-f", "%Sm", str(path)]).decode().strip()


def cmd_list(args: argparse.Namespace) -> int:
    pdir = project_dir()
    if not pdir.exists():
        print(f"No transcript directory: {pdir}", file=sys.stderr)
        return 1
    files = sorted(pdir.glob("*.jsonl"), key=lambda p: p.stat().st_mtime, reverse=True)
    if not files:
        print(f"No JSONL transcripts in {pdir}", file=sys.stderr)
        return 1
    for p in files[: args.limit]:
        print(f"{p}")
        print(f"  modified: {mtime(p)}")
        print(f"  first:    {first_user_text(p)}")
        print()
    return 0


def cmd_extract(args: argparse.Namespace) -> int:
    src = Path(args.path)
    if not src.exists():
        print(f"Not found: {src}", file=sys.stderr)
        return 1
    out = Path(args.out) if args.out else Path("/tmp/recovered-session.md")
    turns = 0
    with src.open() as f, out.open("w") as o:
        for line in f:
            try:
                msg = json.loads(line)
            except json.JSONDecodeError:
                continue
            msg_type = msg.get("type")
            content = msg.get("message", {}).get("content", "")

            text = ""
            if isinstance(content, str):
                text = content.strip()
            elif isinstance(content, list):
                texts = [
                    b.get("text", "")
                    for b in content
                    if isinstance(b, dict) and b.get("type") == "text"
                ]
                text = " ".join(t for t in texts if t).strip()

            if not text:
                continue
            if msg_type == "user":
                o.write(f"## User\n{text[: args.max_chars]}\n\n")
                turns += 1
            elif msg_type == "assistant":
                o.write(f"## Assistant\n{text[: args.max_chars]}\n\n")
                turns += 1

    print(f"Wrote {turns} turns to {out}")
    return 0


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)

    p_list = sub.add_parser("list", help="list recent transcripts")
    p_list.add_argument("--limit", type=int, default=5)
    p_list.set_defaults(func=cmd_list)

    p_ext = sub.add_parser("extract", help="extract human/assistant turns")
    p_ext.add_argument("path", help="path to JSONL transcript")
    p_ext.add_argument("--out", help="output path (default /tmp/recovered-session.md)")
    p_ext.add_argument("--max-chars", type=int, default=500, help="truncate each turn (default 500)")
    p_ext.set_defaults(func=cmd_extract)

    args = p.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
