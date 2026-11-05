#!/usr/bin/env python3
"""Features list: docs/features.md is generated from docs/features.tsv.

    python3 tools/features.py            # write docs/features.md
    python3 tools/features.py --check    # write nothing; exit 1 on any problem listed below
    python3 tools/features.py --root DIR # use another repository root (the tests use this)

docs/features.tsv is plain tab-separated text: the first row is the header, every row has the same 8 cells, and no
cell holds a straight double quote (GitHub's table view of a .tsv reads one as CSV quoting). Columns:

    id        short, stable, lower case with at least one dash (rooms-basic); never reused for another feature
    area      grouping on the page (same text = same table; tables appear in the order of first use)
    feature   short name
    detail    what it does, in a sentence or two
    status    available    merged, with a test or a recorded run in this repository
              unverified   merged, but no test or recorded run covers it (the note says what was not checked)
              partial      merged in part, or works with known gaps
              in-progress  an open pull request or branch
              planned      wanted and intended, not started
              later        wanted, after the planned items or after a decision
              declined     will not be done
    since     the release that first had it (a version from CHANGELOG.md), main for merged but not yet released,
              empty for anything not merged
    evidence  comma-separated: a repository path (must exist), a Go test or fuzz name (TestX, FuzzX; must be defined
              in a *_test.go file), or a pull request of this repository (#7). Required for available.
    note      short remark, optional

--check fails when: docs/features.md is missing or not what the data generates; the header is wrong; a row has the
wrong number of cells; a status is unknown; an id is malformed or used twice; an available row has no evidence; an
evidence path or test name does not exist; a markdown link in a cell points at a missing file; a cell holds a straight
double quote; or ROADMAP.md uses an [id] tag that is not in the data.
"""
import argparse
import io
import pathlib
import re
import sys

DATA = "docs/features.tsv"
PAGE = "docs/features.md"
ROADMAP = "ROADMAP.md"
REPO_URL = "https://github.com/undecuple/re-psn"
HEADER = "<!-- Generated from docs/features.tsv; do not edit by hand. -->"
COLUMNS = ("id", "area", "feature", "detail", "status", "since", "evidence", "note")

# status -> (word shown on the page, meaning); the order is the order of the summary.
STATUS = {
    "available": ("available", "merged, with a test or a recorded run"),
    "unverified": ("not yet verified", "merged, but nothing in the repository checks it"),
    "partial": ("partial", "merged in part, or with known gaps"),
    "in-progress": ("in progress", "an open pull request or branch"),
    "planned": ("planned", "wanted and intended, not started"),
    "later": ("later", "after the planned items, or after a decision"),
    "declined": ("declined", "will not be done"),
}

ID_RE = re.compile(r"^[a-z][a-z0-9]*(?:-[a-z0-9]+)+$")
PR_RE = re.compile(r"^#([0-9]+)$")
TEST_RE = re.compile(r"^(?:Test|Fuzz)[A-Za-z0-9_]*$")
LINK_RE = re.compile(r"\]\(([^)\s]+)\)")
# A roadmap tag: [some-id] that is not the text of a markdown link.
TAG_RE = re.compile(r"\[([a-z][a-z0-9]*(?:-[a-z0-9]+)+)\](?!\()")


def read_rows(root, problems):
    """Rows of the data file as dicts (with 'line'), in file order. Malformed rows are reported and skipped."""
    text = (root / DATA).read_text(encoding="utf-8")
    lines = [(i + 1, l) for i, l in enumerate(text.splitlines()) if l.strip()]
    if not lines:
        problems.append(f"{DATA}: empty")
        return []
    head = lines[0][1].split("\t")
    if tuple(head) != COLUMNS:
        problems.append(f"{DATA}:1: header must be {' '.join(COLUMNS)}, got {' '.join(head)}")
        return []
    rows = []
    seen = {}
    for ln, l in lines[1:]:
        cells = l.split("\t")
        if len(cells) != len(COLUMNS):
            problems.append(f"{DATA}:{ln}: {len(cells)} cells, expected {len(COLUMNS)}")
            continue
        if '"' in l:
            problems.append(f"{DATA}:{ln}: straight double quote in a cell; use ' or a typographic quote")
        row = dict(zip(COLUMNS, (c.strip() for c in cells)))
        row["line"] = ln
        if not ID_RE.match(row["id"]):
            problems.append(f"{DATA}:{ln}: id '{row['id']}' must be lower case words joined by dashes")
        elif row["id"] in seen:
            problems.append(f"{DATA}:{ln}: id '{row['id']}' already used on line {seen[row['id']]}")
        else:
            seen[row["id"]] = ln
        if row["status"] not in STATUS:
            problems.append(f"{DATA}:{ln}: status '{row['status']}' not in {', '.join(STATUS)}")
        if not row["feature"]:
            problems.append(f"{DATA}:{ln}: empty feature")
        if not row["area"]:
            problems.append(f"{DATA}:{ln}: empty area")
        if row["status"] == "available" and not row["evidence"]:
            problems.append(f"{DATA}:{ln}: available needs evidence")
        rows.append(row)
    return rows


def test_names(root):
    """Go test and fuzz function name -> file (relative to root), from every *_test.go in the repository."""
    names = {}
    for p in sorted(root.rglob("*_test.go")):
        rel = p.relative_to(root).as_posix()
        for m in re.finditer(r"^func ((?:Test|Fuzz)\w*)\(", p.read_text(encoding="utf-8"), re.M):
            names.setdefault(m.group(1), rel)
    return names


def page_link(path):
    """A repository path as a link target from docs/features.md."""
    return path[len("docs/"):] if path.startswith("docs/") else "../" + path


def cell(s):
    return s.replace("|", "\\|")


def render_evidence(row, root, tests, problems):
    out = []
    for tok in (t.strip() for t in row["evidence"].split(",")):
        if not tok:
            continue
        if (m := PR_RE.match(tok)):
            out.append(f"[#{m.group(1)}]({REPO_URL}/pull/{m.group(1)})")
        elif TEST_RE.match(tok):
            if tok in tests:
                out.append(f"[`{tok}`]({page_link(tests[tok])})")
            else:
                problems.append(f"{DATA}:{row['line']}: evidence test {tok} is not defined in any *_test.go")
                out.append(f"`{tok}`")
        else:
            if (root / tok).exists():
                out.append(f"[`{tok}`]({page_link(tok)})")
            else:
                problems.append(f"{DATA}:{row['line']}: evidence path {tok} does not exist")
                out.append(f"`{tok}`")
    return ", ".join(out)


def check_links(row, root, problems):
    """Markdown links in detail and note are written from the repository root; each must name an existing file."""
    for col in ("detail", "note"):
        for m in LINK_RE.finditer(row[col]):
            t = m.group(1)
            if re.match(r"(https?:|mailto:|#)", t):
                continue
            if not (root / t.split("#")[0]).exists():
                problems.append(f"{DATA}:{row['line']}: link to {t} in {col}, which does not exist")


def relink(text):
    """Rewrite root-relative markdown links for the page's location in docs/."""
    def rel(m):
        t = m.group(1)
        if re.match(r"(https?:|mailto:|#)", t):
            return m.group(0)
        return "](" + page_link(t) + ")"
    return LINK_RE.sub(rel, text)


def counts(rows):
    return {s: sum(1 for r in rows if r["status"] == s) for s in STATUS}


def render(rows, root, problems):
    tests = test_names(root)
    n = counts(rows)
    o = io.StringIO()
    w = o.write
    w(HEADER + "\n\n")
    w("# Features\n\n")
    w("What re-psn does today and what is planned, one row per feature. Each status rests on the evidence next to it: "
      "a test in this repository, a recorded run in the docs, or the pull request that added it. *Not yet verified* "
      "means the code or config is in the repository but nothing here checks it. *Since* is the version that first "
      "had the feature; *main* means merged but not in a release yet.\n\n")
    w("| Status | Meaning | Features |\n|---|---|---|\n")
    for s, (word, meaning) in STATUS.items():
        w(f"| {word} | {meaning} | {n[s]} |\n")
    w(f"| all | | {len(rows)} |\n\n")
    areas = []
    for r in rows:
        if r["area"] not in areas:
            areas.append(r["area"])
    for a in areas:
        w(f"## {a}\n\n| ID | Feature | Status | Since | Evidence | Note |\n|---|---|---|---|---|---|\n")
        for r in (x for x in rows if x["area"] == a):
            check_links(r, root, problems)
            word = STATUS.get(r["status"], (r["status"], ""))[0]
            feat = f"**{cell(r['feature'])}**" + (f": {cell(relink(r['detail']))}" if r["detail"] else "")
            ev = render_evidence(r, root, tests, problems)
            w(f"| `{r['id']}` | {feat} | {word} | {cell(r['since'])} | {cell(ev)} | {cell(relink(r['note']))} |\n")
        w("\n")
    w("The order of the planned work is in [ROADMAP.md](../ROADMAP.md).\n")
    return o.getvalue()


def check_roadmap(rows, root, problems):
    """Every [id] tag in ROADMAP.md must be an id in the data. Returns the number of tags found."""
    p = root / ROADMAP
    if not p.exists():
        return 0
    ids = {r["id"] for r in rows}
    tags = TAG_RE.findall(p.read_text(encoding="utf-8"))
    for t in sorted(set(tags)):
        if t not in ids:
            problems.append(f"{ROADMAP}: tag [{t}] is not an id in {DATA}")
    return len(tags)


def main(argv=None):
    ap = argparse.ArgumentParser(description="Generate docs/features.md from docs/features.tsv, or check it.")
    ap.add_argument("--check", action="store_true", help="write nothing; exit 1 on any problem")
    ap.add_argument("--root", default=str(pathlib.Path(__file__).resolve().parents[1]), help="repository root")
    a = ap.parse_args(argv)
    root = pathlib.Path(a.root)
    problems = []
    try:
        rows = read_rows(root, problems)
    except OSError as e:
        print(f"features: {e}", file=sys.stderr)
        return 1
    page = render(rows, root, problems)
    tags = check_roadmap(rows, root, problems)
    out = root / PAGE
    if a.check:
        if not out.exists():
            problems.append(f"{PAGE} is missing (run python3 tools/features.py)")
        elif out.read_text(encoding="utf-8") != page:
            problems.append(f"{PAGE} is not what {DATA} generates (run python3 tools/features.py)")
    if problems:
        print("features: problems:", file=sys.stderr)
        for p in problems:
            print("  - " + p, file=sys.stderr)
        return 1
    if not a.check:
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(page, encoding="utf-8")
    n = counts(rows)
    summary = ", ".join(f"{s} {n[s]}" for s in STATUS if n[s])
    roadmap = f"{tags} [id] tags in {ROADMAP}, all known" if tags else f"no [id] tags in {ROADMAP} yet"
    print(f"features: {'OK' if a.check else 'wrote ' + PAGE}: {len(rows)} rows ({summary}); {roadmap}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
