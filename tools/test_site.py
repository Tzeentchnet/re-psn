"""Tests for tools/site.py: escaping, deterministic output, --check on a stale or missing file, the stylesheet tokens.

    python3 -m unittest            # from the repository root

The module is loaded by path under another name because "site" is also a standard library module.
"""
import contextlib
import importlib.util
import io
import json
import pathlib
import sys
import tempfile
import unittest

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent
sys.path.insert(0, str(HERE))
_spec = importlib.util.spec_from_file_location("re_psn_site", HERE / "site.py")
sitegen = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(sitegen)
import features  # noqa: E402

HEAD = "\t".join(features.COLUMNS)
ROWS = [
    "rooms-basic\tRooms\tRooms <b>&\tCreate and join rooms, see `x<y>` and [the roadmap](ROADMAP.md).\tavailable\t0.1.0\tserver/rooms.go\tNote 'q'",
    "rooms-private\tRooms\tPrivate rooms\tPasswords.\tplanned\t\t\t",
    "net-relay\tNetwork\tRelay\tThrough the server.\tunverified\tmain\tserver/relay.go\t",
]
README = """# t

| Flag | Variable | Default | What it does |
|---|---|---|---|
| `-addr` | `RE_PSN_ADDR` | `:36100` | TCP and UDP address <script> |
| `-http` | `RE_PSN_HTTP` | `127.0.0.1:36101` | Status page |

after the table
"""
CHANGELOG = """# Changelog

## [Unreleased]

### Added
- A thing. More words.

## [0.1.0] - 2026-01-02

### Added
- First.
"""
CSS = """/* palette:dark */
:root { --bg0: #000000; --fg0: #ffffff; }
/* /palette */
@media (prefers-color-scheme: light) {
/* palette:light */
:root { --bg0: #ffffff; --fg0: #000000; }
/* /palette */
}
"""
PALETTE = {"modes": {"dark": {"bg0": "#000000", "fg0": "#ffffff"}, "light": {"bg0": "#ffffff", "fg0": "#000000"}}}


class Fixture:
    def __init__(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.tmp.name)
        for d in ("docs/art", "site", "server"):
            (self.root / d).mkdir(parents=True)
        for path, _, _ in sitegen.DOCS:
            p = self.root / path
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text("x\n")
        (self.root / "server" / "rooms.go").write_text("package main\n")
        (self.root / "server" / "relay.go").write_text("package main\n")
        (self.root / features.DATA).write_text("\n".join([HEAD] + ROWS) + "\n", encoding="utf-8")
        (self.root / "VERSION").write_text("0.1.0\n")
        (self.root / "README.md").write_text(README)
        (self.root / "CHANGELOG.md").write_text(CHANGELOG)
        (self.root / "go.mod").write_text("module x\n\ngo 1.22\n")
        (self.root / "docs/art/palette.json").write_text(json.dumps(PALETTE))
        (self.root / "site/style.css").write_text(CSS)
        for path, _, _ in sitegen.DOCS:
            p = self.root / path
            p.parent.mkdir(parents=True, exist_ok=True)
            if not p.exists():
                p.write_text("x\n")

    def run(self, *args):
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            code = sitegen.main(["--root", str(self.root), *args])
        return code, out.getvalue(), err.getvalue()

    def read(self, rel):
        return (self.root / rel).read_bytes()


class SiteTest(unittest.TestCase):
    def setUp(self):
        self.f = Fixture()
        self.addCleanup(self.f.tmp.cleanup)

    def test_writes_then_check_passes(self):
        self.assertEqual(self.f.run()[0], 0)
        code, out, err = self.f.run("--check")
        self.assertEqual((code, err), (0, ""), out)

    def test_output_is_deterministic(self):
        self.f.run()
        first = (self.f.read("site/index.html"), self.f.read("site/data.json"))
        self.f.run()
        self.assertEqual(first, (self.f.read("site/index.html"), self.f.read("site/data.json")))

    def test_escaping(self):
        self.f.run()
        page = self.f.read("site/index.html").decode()
        self.assertIn("Rooms &lt;b&gt;&amp;", page)
        self.assertNotIn("<b>", page)
        self.assertIn("<code>x&lt;y&gt;</code>", page)
        self.assertNotIn("<script>", page)
        self.assertIn("TCP and UDP address &lt;script&gt;", page)

    def test_repository_links_and_csp(self):
        self.f.run()
        page = self.f.read("site/index.html").decode()
        self.assertIn(f'<a href="{features.REPO_URL}/blob/main/ROADMAP.md">the roadmap</a>', page)
        self.assertIn("connect-src &#x27;none&#x27;", page)
        self.assertNotIn("<script", page)
        self.assertEqual(page.count("<title>"), 1)

    def test_data_comes_from_sources(self):
        self.f.run()
        d = json.loads(self.f.read("site/data.json"))
        self.assertEqual(d["version"], "0.1.0")
        self.assertEqual(d["status_counts"]["available"], 1)
        self.assertEqual([s["flag"] for s in d["settings"]], ["-addr", "-http"])
        self.assertEqual(d["roadmap"]["Done"], ["rooms-basic", "net-relay"])
        self.assertEqual(d["changelog"]["unreleased"]["sections"]["Added"], ["A thing."])
        self.assertEqual(d["changelog"]["latest"]["version"], "0.1.0")

    def test_check_fails_on_stale_file(self):
        self.f.run()
        (self.f.root / "VERSION").write_text("0.2.0\n")
        code, _, err = self.f.run("--check")
        self.assertEqual(code, 1)
        self.assertIn("site/index.html is not what the sources generate", err)
        self.assertIn("site/data.json is not what the sources generate", err)

    def test_check_fails_on_hand_edit(self):
        self.f.run()
        p = self.f.root / "site/index.html"
        p.write_bytes(p.read_bytes() + b"<!-- x -->\n")
        code, _, err = self.f.run("--check")
        self.assertEqual(code, 1)
        self.assertIn("site/index.html is not what", err)

    def test_check_fails_on_missing_file(self):
        code, _, err = self.f.run("--check")
        self.assertEqual(code, 1)
        self.assertIn("site/index.html is missing", err)

    def test_check_writes_nothing(self):
        self.f.run("--check")
        self.assertFalse((self.f.root / "site/index.html").exists())

    def test_css_token_mismatch(self):
        (self.f.root / "site/style.css").write_text(CSS.replace("#ffffff; }\n/* /palette */\n@media", "#fffffe; }\n/* /palette */\n@media"))
        code, _, err = self.f.run("--check")
        self.assertEqual(code, 1)
        self.assertIn("--fg0 in the dark block", err)

    def test_css_must_not_load_remote(self):
        (self.f.root / "site/style.css").write_text(CSS + "@import url(https://x.example/a.css);\n")
        code, _, err = self.f.run("--check")
        self.assertEqual(code, 1)
        self.assertIn("another site", err)

    def test_forbidden_word_stops_the_build(self):
        rows = ROWS + ["ha-x\tRooms\tHandled by AI\tText.\tplanned\t\t\t"]
        (self.f.root / features.DATA).write_text("\n".join([HEAD] + rows) + "\n", encoding="utf-8")
        code, _, err = self.f.run()
        self.assertEqual(code, 1)
        self.assertIn("'AI'", err)
        self.assertFalse((self.f.root / "site/index.html").exists())

    def test_missing_settings_table(self):
        (self.f.root / "README.md").write_text("# t\n")
        code, _, err = self.f.run()
        self.assertEqual(code, 1)
        self.assertIn("no settings table", err)

    def test_real_repository_is_current(self):
        # The same check CI runs, so a stale site/ fails here too.
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            code = sitegen.main(["--check", "--root", str(ROOT)])
        self.assertEqual(code, 0, err.getvalue())


if __name__ == "__main__":
    unittest.main()
