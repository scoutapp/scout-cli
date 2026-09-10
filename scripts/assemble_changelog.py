#!/usr/bin/env python3
"""assemble_changelog.py — fold changelog.d/ fragments into CHANGELOG.md.

Reads every fragment in changelog.d/ (except README.md), merges their
### Added / ### Changed / ### Fixed / ### Removed sections (in that order,
de-duplicating identical bullets), writes the result into CHANGELOG.md as a
new versioned section right after the "# Changelog" heading, and deletes
the fragment files it consumed.

Usage:
    python3 scripts/assemble_changelog.py X.Y.Z [--date YYYY-MM-DD]

Run this as part of cutting a release (see .claude/commands/release.md).
Review the resulting CHANGELOG.md diff before committing — this script
does not commit anything itself.
"""
import argparse
import datetime
import re
import sys
from pathlib import Path

ORDER = ["Added", "Changed", "Fixed", "Removed"]
REPO_ROOT = Path(__file__).resolve().parent.parent
FRAGMENTS_DIR = REPO_ROOT / "changelog.d"
CHANGELOG_PATH = REPO_ROOT / "CHANGELOG.md"


def parse_fragment(text):
    """Return {heading: [bullet_lines]} from one fragment's text."""
    sections = {}
    current = None
    for line in text.splitlines():
        m = re.match(r"^### (\w+)", line)
        if m:
            current = m.group(1)
            sections.setdefault(current, [])
            continue
        if current and line.strip().startswith("-"):
            sections[current].append(line.rstrip())
    return sections


def render(sections):
    out = []
    for heading in ORDER:
        bullets = sections.get(heading)
        if not bullets:
            continue
        out.append(f"### {heading}\n\n")
        out.extend(b + "\n" for b in bullets)
        out.append("\n")
    # Preserve any non-standard heading at the end rather than silently
    # dropping it.
    for heading, bullets in sections.items():
        if heading not in ORDER and bullets:
            out.append(f"### {heading}\n\n")
            out.extend(b + "\n" for b in bullets)
            out.append("\n")
    return "".join(out)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("version", help="Version being released, e.g. 0.5.0")
    parser.add_argument(
        "--date",
        default=datetime.date.today().isoformat(),
        help="Release date (default: today, YYYY-MM-DD)",
    )
    args = parser.parse_args()

    fragment_paths = sorted(
        p for p in FRAGMENTS_DIR.glob("*.md") if p.name != "README.md"
    )
    if not fragment_paths:
        print("No changelog fragments found in changelog.d/ — nothing to do.")
        sys.exit(0)

    merged = {}
    for path in fragment_paths:
        sections = parse_fragment(path.read_text())
        if not sections:
            print(f"warning: {path} has no ### heading / bullet content, skipping", file=sys.stderr)
            continue
        for heading, bullets in sections.items():
            existing = merged.setdefault(heading, [])
            for b in bullets:
                if b not in existing:
                    existing.append(b)

    if not merged:
        print("Fragments found but none contained usable entries — nothing to do.")
        sys.exit(0)

    body = render(merged)
    new_section = f"## [{args.version}] - {args.date}\n\n{body}"

    changelog = CHANGELOG_PATH.read_text()
    lines = changelog.splitlines(keepends=True)
    # Insert right after the "# Changelog" heading (and the blank line after
    # it, if present), before the first existing "## " section.
    insert_at = 1
    while insert_at < len(lines) and not lines[insert_at].startswith("## "):
        insert_at += 1
    new_changelog = "".join(lines[:insert_at]) + new_section + "".join(lines[insert_at:])
    CHANGELOG_PATH.write_text(new_changelog)

    for path in fragment_paths:
        path.unlink()

    print(f"Wrote ## [{args.version}] - {args.date} to {CHANGELOG_PATH.relative_to(REPO_ROOT)}")
    print(f"Consumed {len(fragment_paths)} fragment(s):")
    for path in fragment_paths:
        print(f"  - {path.relative_to(REPO_ROOT)}")
    print("\nReview the CHANGELOG.md diff, then commit both the changelog and the fragment deletions together.")


if __name__ == "__main__":
    main()
