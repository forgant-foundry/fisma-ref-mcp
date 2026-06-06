"""
Parse the NIST CSF 2.0 to SP 800-53 Rev 5.2.0 crosswalk Excel file into JSON.

Usage:
    python3 tools/parse-csf-crosswalk/parse.py

Output: internal/nist_csf/data/csf-800-53-crosswalk.json
"""

import json
import re
import sys
from collections import defaultdict
from pathlib import Path

try:
    import openpyxl
except ImportError:
    sys.exit("openpyxl required: pip install openpyxl")

REPO = Path(__file__).parent.parent.parent
INPUT = REPO / "internal/nist_csf/data/Cybersecurity_Framework_v2-0_Concept_Crosswalk_800-53_5_2_0_draft.xlsx"
OUTPUT = REPO / "internal/nist_csf/data/csf-800-53-crosswalk.json"

# Matches a SP 800-53 control ID, possibly zero-padded, possibly with enhancement.
CTRL_RE = re.compile(r'^([A-Z]{2})-(\d+)(?:\((\d+)\))?$')
# Matches a CSF 2.0 subcategory ID.
SUB_RE = re.compile(r'^[A-Z]{2}\.[A-Z]{2,3}-\d{2}$')


def normalize_control(raw: str) -> str | None:
    """Normalize a zero-padded SP 800-53 control ID to display form.

    'AC-01' → 'AC-1', 'CP-02(08)' → 'CP-2(8)', 'PT' → None (skip).
    """
    raw = raw.strip().upper()
    m = CTRL_RE.match(raw)
    if not m:
        return None  # bare family, section ref, etc.
    family = m.group(1)
    num = str(int(m.group(2)))
    if m.group(3) is not None:
        enh = str(int(m.group(3)))
        return f"{family}-{num}({enh})"
    return f"{family}-{num}"


def main():
    wb = openpyxl.load_workbook(INPUT, read_only=True)
    ws = wb['Relationships']

    mappings: dict[str, list[str]] = defaultdict(list)
    seen: dict[str, set] = defaultdict(set)
    skipped = 0

    for i, row in enumerate(ws.iter_rows(values_only=True)):
        if i == 0:
            continue  # header

        focal = str(row[0]).strip() if row[0] else ''
        ctrl_raw = str(row[2]).strip() if row[2] else ''

        if not focal or not SUB_RE.match(focal):
            continue  # function or category row, not a subcategory

        if not ctrl_raw:
            continue

        ctrl = normalize_control(ctrl_raw)
        if ctrl is None:
            skipped += 1
            continue

        if ctrl not in seen[focal]:
            seen[focal].add(ctrl)
            mappings[focal].append(ctrl)

    # Sort for deterministic output
    out = {k: sorted(v) for k, v in sorted(mappings.items())}

    result = {
        "source": "NIST Cybersecurity Framework v2.0 to SP 800-53 Rev 5.2.0 Concept Crosswalk (draft)",
        "subcategory_count": len(out),
        "link_count": sum(len(v) for v in out.values()),
        "mappings": out,
    }

    OUTPUT.write_text(json.dumps(result, indent=2))
    print(f"Wrote {OUTPUT}")
    print(f"  {result['subcategory_count']} subcategories, {result['link_count']} links, {skipped} non-control refs skipped")


if __name__ == "__main__":
    main()
