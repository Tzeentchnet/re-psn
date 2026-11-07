"""Run Go JSON tests and require named conformance tests to pass.

Local test skips remain useful without optional compilers or server source.
CI uses this wrapper to reject skipped or filtered-out required tests.
Unrelated platform-specific skips are allowed.
"""

import argparse
import json
import subprocess
import sys


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--require", action="append", required=True, metavar="PACKAGE:TEST")
    parser.add_argument("command", nargs=argparse.REMAINDER,
                        help="-- go test -json [other go test options]")
    args = parser.parse_args(argv)
    required = set()
    for spec in args.require:
        package, sep, test = spec.rpartition(":")
        if not sep or not package or not test:
            parser.error("--require must be PACKAGE:TEST")
        required.add((package, test))
    command = args.command
    if command[:1] == ["--"]:
        command = command[1:]
    if not command:
        parser.error("provide a command after --")
    try:
        child = subprocess.Popen(command, stdout=subprocess.PIPE, text=True,
                                 encoding="utf-8", errors="replace")
    except OSError as exc:
        print(f"conformance: cannot start command: {exc}", file=sys.stderr)
        return 1

    passed, rejected = set(), set()
    invalid = False
    with child.stdout:
        for line in child.stdout:
            try:
                event = json.loads(line)
                if not isinstance(event, dict) or not isinstance(event.get("Action"), str):
                    raise ValueError("not a Go test event")
                if any(field in event and not isinstance(event[field], str)
                       for field in ("Package", "Test")):
                    raise ValueError("invalid package or test field")
                output = event.get("Output", "")
                if not isinstance(output, str):
                    raise ValueError("invalid Output field")
            except (ValueError, TypeError):
                invalid = True
                print("conformance: invalid Go JSON test output", file=sys.stderr)
                continue
            print(output, end="", flush=True)
            key = (event.get("Package"), event.get("Test"))
            if key in required:
                if event["Action"] == "pass":
                    passed.add(key)
                elif event["Action"] in ("skip", "fail"):
                    rejected.add(key)
    status = child.wait()
    missing = required - passed - rejected
    for package, test in sorted(rejected):
        print(f"conformance: required test skipped or failed: {package}:{test}", file=sys.stderr)
    for package, test in sorted(missing):
        print(f"conformance: required test did not pass: {package}:{test}", file=sys.stderr)
    if status:
        print(f"conformance: test command exited {status}", file=sys.stderr)
    if status or invalid or rejected or missing:
        return 1
    print(f"conformance: all {len(required)} required tests passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
