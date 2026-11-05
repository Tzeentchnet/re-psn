"""Tests for tools/features.py: one good fixture, and one broken fixture per rule that --check enforces.

    python3 -m unittest            # from the repository root
"""
import contextlib
import io
import pathlib
import sys
import tempfile
import unittest

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import features  # noqa: E402

HEAD = "\t".join(features.COLUMNS)
GOOD_ROWS = [
    "rooms-basic\tRooms\tRooms\tCreate and join rooms.\tavailable\t0.1.0\tserver/rooms.go, TestRooms, #7\tUpstream work.",
    "rooms-private\tRooms\tPrivate rooms\tPasswords, see [the roadmap](ROADMAP.md).\tplanned\t\t\t",
    "net-relay\tNetwork\tRelay\tThrough the server.\tunverified\tmain\tserver/relay.go\tNot run.",
]


class Fixture:
    def __init__(self, rows=None, roadmap="# Roadmap\n\nSee [ps3recomp#1](https://example.org/1).\n"):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.tmp.name)
        (self.root / "docs").mkdir()
        (self.root / "server").mkdir()
        (self.root / "server" / "rooms.go").write_text("package main\n")
        (self.root / "server" / "relay.go").write_text("package main\n")
        (self.root / "server" / "rooms_test.go").write_text("package main\n\nfunc TestRooms(t *testing.T) {}\n")
        if roadmap is not None:
            (self.root / "ROADMAP.md").write_text(roadmap)
        self.write(GOOD_ROWS if rows is None else rows)

    def write(self, rows):
        (self.root / features.DATA).write_text("\n".join([HEAD] + rows) + "\n", encoding="utf-8")

    def run(self, *args):
        err = io.StringIO()
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
            rc = features.main(["--root", str(self.root), *args])
        return rc, err.getvalue()

    def close(self):
        self.tmp.cleanup()


class FeaturesTest(unittest.TestCase):
    def fixture(self, rows=None, **kw):
        f = Fixture(rows, **kw)
        self.addCleanup(f.close)
        return f

    def broken(self, rows, needle, **kw):
        """Generate from the good data, then swap in the broken rows: --check must fail and name the problem."""
        f = self.fixture(**kw)
        self.assertEqual(f.run()[0], 0)
        f.write(rows)
        rc, err = f.run("--check")
        self.assertEqual(rc, 1, err)
        self.assertIn(needle, err)

    def test_good(self):
        f = self.fixture()
        self.assertEqual(f.run()[0], 0)
        rc, err = f.run("--check")
        self.assertEqual(rc, 0, err)
        page = (f.root / features.PAGE).read_text()
        self.assertTrue(page.startswith(features.HEADER))
        self.assertIn("## Rooms", page)
        self.assertIn("## Network", page)
        self.assertIn("| not yet verified |", page)
        self.assertIn("[`server/rooms.go`](../server/rooms.go)", page)
        self.assertIn("[`TestRooms`](../server/rooms_test.go)", page)
        self.assertIn("(https://github.com/undecuple/re-psn/pull/7)", page)
        self.assertIn("[the roadmap](../ROADMAP.md)", page)

    def test_roadmap_tags_known(self):
        f = self.fixture(roadmap="# Roadmap\n\n- [rooms-private] Private rooms.\n")
        self.assertEqual(f.run()[0], 0)
        self.assertEqual(f.run("--check")[0], 0)

    def test_missing_page(self):
        rc, err = self.fixture().run("--check")
        self.assertEqual(rc, 1)
        self.assertIn("missing", err)

    def test_stale_page(self):
        rows = list(GOOD_ROWS)
        rows[0] = rows[0].replace("Create and join rooms.", "Create, join and leave rooms.")
        self.broken(rows, "is not what")

    def test_hand_edited_page(self):
        f = self.fixture()
        f.run()
        page = f.root / features.PAGE
        page.write_text(page.read_text() + "extra\n")
        rc, err = f.run("--check")
        self.assertEqual(rc, 1)
        self.assertIn("is not what", err)

    def test_bad_header(self):
        f = self.fixture()
        f.run()
        (f.root / features.DATA).write_text("id\tarea\n" + "\n".join(GOOD_ROWS) + "\n")
        rc, err = f.run("--check")
        self.assertEqual(rc, 1)
        self.assertIn("header", err)

    def test_cell_count(self):
        self.broken(GOOD_ROWS + ["net-extra\tNetwork\tExtra\tplanned\t\t"], "cells, expected 8")

    def test_unknown_status(self):
        self.broken(GOOD_ROWS + ["net-extra\tNetwork\tExtra\tSomething.\tdone\t\t\t"], "status 'done'")

    def test_duplicate_id(self):
        self.broken(GOOD_ROWS + ["net-relay\tNetwork\tAgain\tSomething.\tplanned\t\t\t"], "already used")

    def test_bad_id(self):
        self.broken(GOOD_ROWS + ["Relay2\tNetwork\tAgain\tSomething.\tplanned\t\t\t"], "id 'Relay2'")

    def test_available_without_evidence(self):
        self.broken(GOOD_ROWS + ["net-extra\tNetwork\tExtra\tSomething.\tavailable\tmain\t\t"], "available needs evidence")

    def test_missing_evidence_path(self):
        self.broken(GOOD_ROWS + ["net-extra\tNetwork\tExtra\tSomething.\tavailable\tmain\tserver/gone.go\t"],
                    "evidence path server/gone.go does not exist")

    def test_missing_evidence_test(self):
        self.broken(GOOD_ROWS + ["net-extra\tNetwork\tExtra\tSomething.\tavailable\tmain\tTestGone\t"],
                    "evidence test TestGone")

    def test_link_to_missing_file(self):
        self.broken(GOOD_ROWS + ["net-extra\tNetwork\tExtra\tSee [this](docs/gone.md).\tplanned\t\t\t"],
                    "link to docs/gone.md")

    def test_straight_double_quote(self):
        self.broken(GOOD_ROWS + ['net-extra\tNetwork\tExtra\tThe "fast" path.\tplanned\t\t\t'], "straight double quote")

    def test_unknown_roadmap_tag(self):
        f = self.fixture()
        self.assertEqual(f.run()[0], 0)
        (f.root / "ROADMAP.md").write_text("# Roadmap\n\n- [rooms-private] Private rooms.\n- [net-gone] Something.\n")
        rc, err = f.run("--check")
        self.assertEqual(rc, 1)
        self.assertIn("tag [net-gone]", err)
        self.assertNotIn("rooms-private", err)

    def test_roadmap_buckets(self):
        f = self.fixture(roadmap="# Roadmap\n\n## Scope\n\n- [net-relay] Mentioned anywhere.\n\n## Done\n\n- [rooms-basic] Rooms.\n"
                                 "- [net-relay] Relay.\n\n### A group\n\n## Next\n\n- [rooms-private] Private rooms.\n")
        self.assertEqual(f.run()[0], 0)
        rc, err = f.run("--check")
        self.assertEqual(rc, 0, err)

    def test_roadmap_wrong_bucket(self):
        f = self.fixture()
        self.assertEqual(f.run()[0], 0)
        (f.root / "ROADMAP.md").write_text("# Roadmap\n\n## Done\n\n- [rooms-basic] Rooms.\n\n### Sub\n\n- [rooms-private] Private.\n"
                                         "\n## Later\n\n- [net-relay] Relay.\n")
        rc, err = f.run("--check")
        self.assertEqual(rc, 1)
        self.assertIn("[rooms-private] is planned, which does not belong under 'done'", err)
        self.assertIn("[net-relay] is unverified, which does not belong under 'later'", err)
        self.assertNotIn("[rooms-basic]", err)


if __name__ == "__main__":
    unittest.main()
