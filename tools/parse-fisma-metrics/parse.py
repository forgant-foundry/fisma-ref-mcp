#!/usr/bin/env python3
"""
Layout-aware parser for FY 2025 IG FISMA Metrics Evaluation Guide PDF.

Uses pdftotext -layout output.  Column boundaries (character positions):

  Case A — criteria present on line (first-non-space < 34):
    col_criteria :  pos 11–33  (criteria / reference bullets)
    col_maturity :  pos 52–94  (maturity-level name + description)
    col_evidence :  pos 95+    (suggested standard source evidence)

  Case B — no criteria on line (first-non-space ≥ 34):
    The review-cycle / maturity / evidence zone collapses leftward to ~38.
    Maturity and evidence are split by finding the first 5+ space gap after 38.

Produces: internal/fisma/data/fy2025-ig-fisma-metrics.json
"""

import json
import os
import re
import subprocess
import sys

# ── paths ───────────────────────────────────────────────────────────────────
REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
PDF = os.path.join(REPO_ROOT, "internal", "fisma", "data",
                   "Final FY 2025 IG FISMA Metrics Evaluation Guide_05 May 2025-508.pdf")
OUT = os.path.join(REPO_ROOT, "internal", "fisma", "data", "fy2025-ig-fisma-metrics.json")

# ── column boundaries ────────────────────────────────────────────────────────
COL_CRITERIA_START = 11
COL_CRITERIA_END   = 34   # criteria text ends at pos 33 (exclusive slice :34)
COL_RIGHT_START    = 34   # review-cycle / maturity zone starts at 34
COL_REVIEW_START   = 34   # review cycle detection zone start
COL_REVIEW_END     = 52   # review cycle detection zone end
COL_MATURITY_START = 52   # maturity name/description start (Case A)
COL_EVIDENCE_START = 95   # evidence column start (Case A)

# ── known values ─────────────────────────────────────────────────────────────
DOMAINS = [
    "Cybersecurity Governance",
    "Cybersecurity Supply Chain Risk Management (C-SCRM)",
    "Risk and Asset Management (RAM)",
    "Configuration Management",
    "Identity and Access Management (IDAM)",
    "Data Protection and Privacy",
    "Security Training",
    "Information Security Continuous Monitoring (ISCM)",
    "Incident Response (IR)",
    "Contingency Planning (CP)",
]

MATURITY_NAMES = [
    "Ad Hoc",
    "Ad-Hoc",      # alternate spelling
    "Defined",
    "Consistently Implemented",
    "Managed and Measurable",
    "Managed and measurable",  # alternate capitalisation
    "Optimized",
    "Optimized:",  # trailing colon variant
]

CANONICAL_MATURITY = {
    "Ad Hoc":                    "Ad Hoc",
    "Ad-Hoc":                    "Ad Hoc",
    "Defined":                   "Defined",
    "Consistently Implemented":  "Consistently Implemented",
    "Consistently Implemented.": "Consistently Implemented",
    "Managed and Measurable":    "Managed and Measurable",
    "Managed and measurable":    "Managed and Measurable",
    "Optimized":                 "Optimized",
    "Optimized:":                "Optimized",
}

REVIEW_CYCLE_KEYWORDS = ["FY 2025", "Annual", "Biennial", "Core", "Supplemental"]

# NIST 800-53 control ID pattern  e.g.  AC-1   PM-11   SA-4(1)
CTRL_ID_RE = re.compile(
    r'\b([A-Z]{2}-\d+(?:\(\d+\))?)\b'
)
NIST_REF_MARKER = re.compile(r'NIST SP 800-53', re.IGNORECASE)

# ── helpers ──────────────────────────────────────────────────────────────────

def run_pdftotext(pdf_path: str) -> list[str]:
    result = subprocess.run(
        ["pdftotext", "-layout", pdf_path, "-"],
        capture_output=True, check=True,
    )
    return result.stdout.decode("utf-8", errors="replace").splitlines()


def col(line: str, start: int, end: int | None = None) -> str:
    """Return characters in [start, end) stripped of trailing whitespace."""
    s = line[start:end] if end else line[start:]
    return s.rstrip()


def has_content(line: str, start: int, end: int | None = None) -> bool:
    return bool(col(line, start, end).strip())


def norm_ws(s: str) -> str:
    """Collapse internal whitespace."""
    return re.sub(r'\s+', ' ', s).strip()


def detect_maturity_in_right(line: str) -> str | None:
    """Return canonical maturity name if a maturity keyword is present at col 34+."""
    right = line[COL_RIGHT_START:] if len(line) > COL_RIGHT_START else ""
    for name in MATURITY_NAMES:
        if name in right:
            return CANONICAL_MATURITY.get(name, name)
    return None


def detect_review_cycle(line: str) -> str | None:
    """Return a review cycle keyword found in the review-cycle zone (34–51)."""
    zone = line[COL_REVIEW_START:COL_REVIEW_END] if len(line) > COL_REVIEW_START else ""
    for kw in REVIEW_CYCLE_KEYWORDS:
        if kw in zone:
            return kw
    return None


def extract_nist_control_ids(text: str) -> list[str]:
    """Find all NIST SP 800-53 control IDs in text."""
    # Normalise split control IDs: "PM- 11" → "PM-11", "PM-\n11" → "PM-11"
    text = re.sub(r'([A-Z]{2})-\s+(\d+)', r'\1-\2', text)
    ids: list[str] = []
    # Only extract IDs that appear after a NIST SP 800-53 marker
    segments = NIST_REF_MARKER.split(text)
    for seg in segments[1:]:
        window = seg[:160]
        found = CTRL_ID_RE.findall(window)
        for ctrl_id in found:
            if ctrl_id not in ids:
                ids.append(ctrl_id)
    return ids


def build_criteria_records(criteria_lines: list[str], metric_id: int) -> list[dict]:
    """
    Parse joined criteria text into individual criterion records.
    Returns list of {ref_type, ref_text, control_ids}.
    """
    full_text = " ".join(norm_ws(l) for l in criteria_lines if l.strip())

    records: list[dict] = []

    # Split into bullet points (split on leading •  or similar)
    # Use regex to split on bullet markers
    bullets = re.split(r'\s*[•·]\s*', full_text)

    for bullet in bullets:
        bullet = norm_ws(bullet)
        if not bullet:
            continue

        ctrl_ids = extract_nist_control_ids(bullet)
        if ctrl_ids:
            ref_type = "nist_800_53"
        elif re.search(r'NIST CSF', bullet, re.IGNORECASE):
            ref_type = "nist_csf"
        elif re.search(r'OMB', bullet, re.IGNORECASE):
            ref_type = "omb"
        elif re.search(r'FISMA', bullet, re.IGNORECASE):
            ref_type = "fisma"
        elif re.search(r'FIPS', bullet, re.IGNORECASE):
            ref_type = "nist_fips"
        elif re.search(r'EO\s+\d|Executive Order', bullet, re.IGNORECASE):
            ref_type = "executive_order"
        elif re.search(r'Green Book|CNSSI|DHS BOD|Federal\s+IT', bullet, re.IGNORECASE):
            ref_type = "other_federal"
        elif bullet.strip():
            ref_type = "other"
        else:
            continue

        records.append({
            "ref_type":    ref_type,
            "ref_text":    bullet,
            "control_ids": ctrl_ids,
        })

    return records


# ── main parser ──────────────────────────────────────────────────────────────

def parse(lines: list[str]) -> list[dict]:
    metrics: list[dict] = []

    current_domain   = ""
    current_metric: dict | None = None
    current_maturity: str | None = None  # Ad Hoc / Defined / etc.
    current_review_cycle: str = ""

    # per-metric accumulators
    criteria_lines:  list[str] = []
    maturity_descs:  dict[str, list[str]] = {}
    evidence_items:  dict[str, list[str]] = {}
    assessor_notes:  dict[str, list[str]] = {}

    # state flags
    in_table        = False   # True after first table data row for current metric
    table_header_seen = False  # True after "Criteria...Maturity Level" header line
    in_assessor     = False
    assessor_level: str | None = None

    # Flush the current metric and append to results
    def flush_metric():
        nonlocal current_metric, current_maturity, current_review_cycle
        nonlocal criteria_lines, maturity_descs, evidence_items, assessor_notes
        nonlocal in_table, table_header_seen, in_assessor, assessor_level

        if current_metric is None:
            return

        # Build criteria records
        criteria_records = build_criteria_records(criteria_lines, current_metric["id"])

        # Build maturity level records
        levels = []
        for lvl_name in ["Ad Hoc", "Defined", "Consistently Implemented",
                          "Managed and Measurable", "Optimized"]:
            desc_lines = maturity_descs.get(lvl_name, [])
            ev_lines   = evidence_items.get(lvl_name, [])
            notes_lines = assessor_notes.get(lvl_name, [])

            desc  = norm_ws(" ".join(desc_lines))
            ev    = norm_ws(" ".join(ev_lines))
            notes = norm_ws(" ".join(notes_lines))

            levels.append({
                "level":          lvl_name,
                "description":    desc,
                "evidence":       ev,
                "assessor_notes": notes,
            })

        current_metric["review_cycle"]   = current_review_cycle
        current_metric["maturity_levels"] = levels
        current_metric["criteria"]       = criteria_records
        metrics.append(current_metric)

        # reset
        current_metric       = None
        current_maturity     = None
        current_review_cycle = ""
        criteria_lines       = []
        maturity_descs       = {}
        evidence_items       = {}
        assessor_notes       = {}
        in_table             = False
        table_header_seen    = False
        in_assessor          = False
        assessor_level       = None

    # ── line-by-line scan ─────────────────────────────────────────────────────
    for raw_line in lines:
        line = raw_line.rstrip('\n')
        stripped = line.strip()

        # Skip page separators and classification headers
        if stripped in ("PUBLIC/OFFICIAL RELEASE // EXTERNAL", ""):
            continue
        # Skip page numbers
        if re.fullmatch(r'\d{1,3}', stripped):
            continue
        # Skip form-feed
        if '\x0c' in line:
            continue

        # ── domain header ─────────────────────────────────────────────────────
        if stripped in DOMAINS:
            flush_metric()
            current_domain    = stripped
            in_table          = False
            table_header_seen = False
            in_assessor       = False
            continue

        # ── metric question ───────────────────────────────────────────────────
        metric_match = re.match(r'\s+(\d{1,2})\.\s+(.+)', line)
        if metric_match and int(metric_match.group(1)) <= 35 and current_domain:
            # Verify it looks like a real question (starts with capital, not a list item)
            question_start = metric_match.group(2).strip()
            if question_start[0].isupper() and not re.match(r'^\d', question_start):
                flush_metric()
                current_metric = {
                    "id":     int(metric_match.group(1)),
                    "domain": current_domain,
                    "question": question_start,
                }
                in_table = True
                table_header_seen = False
                in_assessor = False
                continue

        if current_metric is None:
            continue

        # ── table header detection ────────────────────────────────────────────
        if re.search(r'\bCriteria\b', stripped) and re.search(r'\bMaturity\b', stripped):
            table_header_seen = True
            continue
        if stripped in ("Review", "Cycle", "Maturity Level", "Suggested Standard Source Evidence"):
            continue

        # ── question continuation (before table header is seen) ───────────────
        if in_table and not table_header_seen:
            if stripped and current_maturity is None and "question" in current_metric:
                current_metric["question"] = current_metric["question"] + " " + stripped
            continue

        # ── assessor best practices header ────────────────────────────────────
        if "Assessor Best Practices" in stripped:
            in_assessor = True
            in_table = False
            assessor_level = None
            continue

        # ── assessor notes content ─────────────────────────────────────────────
        if in_assessor:
            # Level label lines at pos 11: "Defined:", "Consistently Implemented:", etc.
            assessor_label = re.match(
                r'\s{8,14}(Defined|Consistently Implemented|Managed and (?:M|m)easurable|Optimized)\s*:\s*(.*)',
                line
            )
            if assessor_label:
                lvl_raw = assessor_label.group(1).strip()
                lvl = CANONICAL_MATURITY.get(lvl_raw, lvl_raw)
                assessor_level = lvl
                rest = assessor_label.group(2).strip()
                if rest:
                    assessor_notes.setdefault(assessor_level, []).append(rest)
                continue

            if assessor_level and stripped:
                # continuation line (indented beyond 11)
                if len(line) > COL_CRITERIA_START and line[COL_CRITERIA_START] != ' ':
                    assessor_notes.setdefault(assessor_level, []).append(stripped)
                elif len(line) > COL_CRITERIA_START + 3:
                    assessor_notes.setdefault(assessor_level, []).append(stripped)
            continue

        # ── table content ─────────────────────────────────────────────────────
        if not in_table or not table_header_seen:
            continue

        fns = len(line) - len(line.lstrip()) if line.strip() else 999

        # ── criteria column: positions 11–33 only ────────────────────────────
        # Content at pos 34+ belongs to the review-cycle / maturity / evidence columns.
        crit_text = line[COL_CRITERIA_START:COL_CRITERIA_END].strip() if len(line) > COL_CRITERIA_START else ""

        # ── detect review cycle ───────────────────────────────────────────────
        rc = detect_review_cycle(line)
        if rc and not current_review_cycle:
            current_review_cycle = rc
        elif rc == "Supplemental" and current_review_cycle == "FY 2025":
            current_review_cycle = "FY 2025 Supplemental"

        # ── detect maturity level change ──────────────────────────────────────
        new_maturity = detect_maturity_in_right(line)
        if new_maturity and new_maturity != current_maturity:
            current_maturity = new_maturity

        # ── extract maturity description and evidence ─────────────────────────
        #
        # Two layout cases depending on whether criteria text is present:
        #
        # Case A (fns < COL_REVIEW_START): criteria present at 11–37.
        #   Maturity description is at 52–94 (the dedicated maturity column).
        #   Evidence is at 95+.
        #
        # Case B (fns >= COL_REVIEW_START): no criteria on this line.
        #   The review-cycle / maturity / evidence columns collapse leftward to pos 38.
        #   We detect the evidence boundary by finding the first 5+ space gap after pos 38.
        #
        mat_text = ""
        ev_text  = ""

        # For both Case A and Case B, extract right-side content starting at COL_RIGHT_START (34).
        # Then strip any leading review-cycle keyword and find the evidence boundary via gap detection.
        right_raw = line[COL_RIGHT_START:] if len(line) > COL_RIGHT_START else ""

        # Remove leading review-cycle keyword (e.g. "FY 2025", "Supplemental")
        right_stripped = right_raw.lstrip()
        for kw in REVIEW_CYCLE_KEYWORDS:
            if right_stripped.startswith(kw):
                right_stripped = right_stripped[len(kw):].lstrip()
                break

        # Use a 5+ space gap to split maturity description from evidence
        gap_m = re.search(r'\S(\s{5,})\S', right_stripped)
        if gap_m:
            mat_end  = gap_m.start() + 1
            ev_start = mat_end + len(gap_m.group(1))
            mat_text = right_stripped[:mat_end].strip()
            ev_text  = right_stripped[ev_start:].strip()
        else:
            mat_text = right_stripped.strip()

        # Strip maturity level name from mat_text
        if new_maturity and mat_text:
            for name in MATURITY_NAMES:
                if mat_text.startswith(name):
                    mat_text = mat_text[len(name):].lstrip(".: \t").strip()
                    break

        # Strip isolated maturity names from ev_text (gap-detection artifact)
        for name in MATURITY_NAMES:
            if ev_text == name or ev_text.startswith(name + " ") or ev_text.startswith(name + "."):
                ev_text = ""
                break

        # ── accumulate criteria ───────────────────────────────────────────────
        if crit_text and not any(crit_text == kw for kw in REVIEW_CYCLE_KEYWORDS):
            criteria_lines.append(crit_text)

        # ── accumulate maturity description ───────────────────────────────────
        if mat_text and current_maturity:
            maturity_descs.setdefault(current_maturity, []).append(mat_text)

        # ── accumulate evidence ───────────────────────────────────────────────
        if ev_text and current_maturity:
            evidence_items.setdefault(current_maturity, []).append(ev_text)

    flush_metric()
    return metrics


# ── entrypoint ────────────────────────────────────────────────────────────────

def main():
    if not os.path.exists(PDF):
        print(f"ERROR: PDF not found: {PDF}", file=sys.stderr)
        sys.exit(1)

    print(f"Extracting text from PDF…", file=sys.stderr)
    lines = run_pdftotext(PDF)
    print(f"  {len(lines)} lines", file=sys.stderr)

    print("Parsing metrics…", file=sys.stderr)
    metrics = parse(lines)
    print(f"  {len(metrics)} metrics found", file=sys.stderr)

    # Sort by metric ID
    metrics.sort(key=lambda m: m["id"])

    # Summary
    for m in metrics:
        nist_refs = sum(
            1 for c in m.get("criteria", []) if c["ref_type"] == "nist_800_53"
        )
        all_ctrl_ids = list({
            cid
            for c in m.get("criteria", [])
            for cid in c.get("control_ids", [])
        })
        print(f"  Metric {m['id']:2d} ({m['domain'][:30]}…): "
              f"{nist_refs} NIST refs, ctrl_ids={all_ctrl_ids[:5]}", file=sys.stderr)

    with open(OUT, "w") as f:
        json.dump(metrics, f, indent=2)
    print(f"\nWritten to {OUT}", file=sys.stderr)
    print(f"Total metrics: {len(metrics)}", file=sys.stderr)


if __name__ == "__main__":
    main()
