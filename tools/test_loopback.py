"""Exercise runner exit handling and owned-process cleanup with real children."""

import contextlib
import io
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest
from unittest import mock

from tools import loopback


class LoopbackTest(unittest.TestCase):
    def run_child(self, script, timeout=5):
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            return loopback.run_command("fixture", [sys.executable, "-c", script], None, timeout)

    def test_success_and_failure_exit(self):
        self.assertEqual(self.run_child("pass"), 0)
        self.assertEqual(self.run_child("raise SystemExit(7)"), 1)

    def test_command_cannot_start(self):
        with tempfile.TemporaryDirectory() as directory:
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                status = loopback.run_command("missing", [str(Path(directory) / "missing")], None, 1)
        self.assertEqual(status, 1)

    def test_timeout_is_failure(self):
        self.assertEqual(self.run_child("import time; time.sleep(60)", timeout=0.1), 1)

    def test_nonpositive_timeout_is_usage_error(self):
        with contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as error:
                loopback.main(["--timeout", "0"])
        self.assertEqual(error.exception.code, 2)

    def test_server_settings_are_not_inherited(self):
        with mock.patch.dict(os.environ, {"RE_PSN_HTTP": "fixture", "psnr_db": "fixture", "LOOPBACK_TEST": "keep"}):
            env = loopback.clean_env()
        self.assertNotIn("RE_PSN_HTTP", env)
        self.assertNotIn("psnr_db", env)
        self.assertEqual(env["LOOPBACK_TEST"], "keep")

    @unittest.skipUnless(sys.platform.startswith("linux"), "uses /proc to check child termination")
    def test_descendant_removed_after_success_or_timeout(self):
        for wait in (False, True):
            with self.subTest(timeout=wait), tempfile.TemporaryDirectory() as directory:
                pid_file = Path(directory) / "pid"
                script = ("import subprocess,sys,time; from pathlib import Path; "
                          "child=subprocess.Popen([sys.executable,'-c','import time; time.sleep(60)']); "
                          f"Path({str(pid_file)!r}).write_text(str(child.pid)); "
                          + ("time.sleep(60)" if wait else "pass"))
                self.assertEqual(self.run_child(script, timeout=1), 1 if wait else 0)
                pid = int(pid_file.read_text())
                stat = Path(f"/proc/{pid}/stat")
                deadline = time.monotonic() + 2
                while stat.exists() and time.monotonic() < deadline:
                    try:
                        if stat.read_text().rsplit(")", 1)[1].split()[0] == "Z":
                            break
                    except (FileNotFoundError, ProcessLookupError):
                        # Linux can return ENOENT or ESRCH if the child exits during the read.
                        break
                    time.sleep(0.01)
                else:
                    self.assertFalse(stat.exists(), "owned descendant survived runner exit")


if __name__ == "__main__":
    unittest.main()
