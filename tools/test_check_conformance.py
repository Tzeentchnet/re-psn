"""Verify enforcement against real child-process JSON test streams."""

import contextlib
import io
import json
import pathlib
import sys
import unittest

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import check_conformance


class ConformanceCheckTest(unittest.TestCase):
    def run_output(self, stream, status=0, required=("example/server:TestCClient",)):
        args = []
        for name in required:
            args.extend(["--require", name])
        args.extend(["--", sys.executable, "-c",
                     f"import sys; sys.stdout.write({stream!r}); sys.exit({status})"])
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            rc = check_conformance.main(args)
        return rc, out.getvalue(), err.getvalue()

    def run_stream(self, events, **kwargs):
        return self.run_output("".join(json.dumps(e) + "\n" for e in events), **kwargs)

    def event(self, action, package="example/server", test="TestCClient", **kwargs):
        return dict(Action=action, Package=package, Test=test, **kwargs)

    def test_all_required_tests_must_pass(self):
        events = [self.event("output", Output="native client passed\n"), self.event("pass"),
                  self.event("pass", "example/app", "TestRooms")]
        rc, out, err = self.run_stream(events, required=(
            "example/server:TestCClient", "example/app:TestRooms"))
        self.assertEqual(rc, 0, err)
        self.assertIn("native client passed", out)
        self.assertIn("all 2 required tests passed", out)

    def test_required_skip_fails_even_when_command_succeeds(self):
        rc, _, err = self.run_stream([self.event("skip")])
        self.assertEqual(rc, 1)
        self.assertIn("required test skipped or failed", err)

    def test_filtered_out_or_renamed_test_fails(self):
        rc, _, err = self.run_stream([self.event("pass", test="TestOther")])
        self.assertEqual(rc, 1)
        self.assertIn("required test did not pass", err)

    def test_other_package_cannot_satisfy_requirement(self):
        rc, _, err = self.run_stream([self.event("pass", package="example/other")])
        self.assertEqual(rc, 1)
        self.assertIn("example/server:TestCClient", err)

    def test_failed_command_is_not_hidden(self):
        rc, _, err = self.run_stream([self.event("pass")], status=7)
        self.assertEqual(rc, 1)
        self.assertIn("exited 7", err)

    def test_unrelated_platform_skip_is_allowed(self):
        rc, _, err = self.run_stream([self.event("skip", test="TestOtherPlatform"), self.event("pass")])
        self.assertEqual(rc, 0, err)

    def test_any_required_skip_in_repeated_runs_fails(self):
        rc, _, err = self.run_stream([self.event("pass"), self.event("skip")])
        self.assertEqual(rc, 1)
        self.assertIn("required test skipped or failed", err)

    def test_malformed_output_fails(self):
        rc, _, err = self.run_output(json.dumps(self.event("pass")) + "\nnot json\n")
        self.assertEqual(rc, 1)
        self.assertIn("invalid Go JSON", err)

    def test_non_event_json_fails(self):
        rc, _, err = self.run_stream([[], self.event("pass")])
        self.assertEqual(rc, 1)
        self.assertIn("invalid Go JSON", err)

    def test_invalid_test_field_fails_without_crashing(self):
        rc, _, err = self.run_stream([self.event("pass", test=[]), self.event("pass")])
        self.assertEqual(rc, 1)
        self.assertIn("invalid Go JSON", err)

    def test_required_failure_is_not_hidden_by_success_exit(self):
        rc, _, err = self.run_stream([self.event("fail")])
        self.assertEqual(rc, 1)
        self.assertIn("required test skipped or failed", err)

    def test_command_that_cannot_start_fails(self):
        with contextlib.redirect_stderr(io.StringIO()) as err:
            rc = check_conformance.main([
                "--require", "example/server:TestCClient", "--",
                str(pathlib.Path(__file__).parent / "no-such-conformance-command"),
            ])
        self.assertEqual(rc, 1)
        self.assertIn("cannot start command", err.getvalue())

    def test_empty_output_fails(self):
        rc, _, err = self.run_output("")
        self.assertEqual(rc, 1)
        self.assertIn("required test did not pass", err)

    def test_missing_command_fails(self):
        with contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as raised:
                check_conformance.main(["--require", "example/server:TestCClient"])
        self.assertEqual(raised.exception.code, 2)


if __name__ == "__main__":
    unittest.main()
