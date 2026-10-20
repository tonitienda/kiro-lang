"""Trusted baseline HTTP parity checks, independent of the six public workload tests.

Run directly with --trusted-fixture --kiro /absolute/kiro. Discovery runs only the
case-generator/authorization checks. Never execute model candidates with this runner.
Arithmetic overflow is outside the documented workload input domain.
"""
import argparse
import contextlib
import json
import io
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile
import time
import unittest
from unittest.mock import patch
import urllib.error
import urllib.request

from evaluate import command, compare_response
from oracle import cases

ROOT = Path(__file__).resolve().parents[1]


def boundary_cases():
    out = []
    def add(path, body, status=400, expected=None, name="schema"):
        out.append(dict(id=f"independent-{name}-{len(out):04d}", group=name,
                        path=path, method="POST", body=body, status=status, expected=expected))
    objects = {
        "/checkout": dict(sku="book", units=1, available=5, delivery="standard", payment="card"),
        "/reservation": dict(now=1, expires_at=3),
        "/refund": dict(unit_cents=700, units=2, returned_units=1, shipping_cents=300),
    }
    for path, body in objects.items():
        for raw in ("null", "[]", "true", "1", '"text"', "", "{", json.dumps(body) + " false"):
            add(path, raw)
        for field, value in body.items():
            without = dict(body); without.pop(field)
            add(path, json.dumps(without))
            for invalid in (None, [], {}, True, 1.5) if isinstance(value, int) else (None, [], {}, True, 1):
                add(path, json.dumps(body | {field: invalid}))
            renamed = dict(without); renamed[field.upper()] = value
            add(path, json.dumps(renamed))
            encoded = json.dumps(body)
            add(path, encoded[:-1] + ',' + json.dumps(field) + ':' + json.dumps(value) + '}')
            if isinstance(value, int):
                for invalid in (-2147483649, 2147483648):
                    add(path, json.dumps(body | {field: invalid}), name="i32-schema")
                for literal in ("1.0", "1e0", '"1"'):
                    raw = json.dumps(body | {field: "PLACEHOLDER"}).replace('"PLACEHOLDER"', literal)
                    add(path, raw, name="i32-schema")
    checkout = objects["/checkout"]
    for coupon in (None, "SHIPFREE"):
        body = checkout | {"coupon": coupon}
        shipping = 300 if coupon is None else 0
        add("/checkout", json.dumps(body), 200,
            dict(sku="book", units=1, remaining_stock=4, subtotal_cents=1500, discount_cents=0,
                 shipping_cents=shipping, fee_cents=30, total_cents=1530 + shipping,
                 payment_reference=f"card:{1500 + shipping}"), "optional")
    for coupon in (0, False, [], {}):
        add("/checkout", json.dumps(checkout | {"coupon": coupon}), name="optional")
    for now, expiry in ((-2147483648, -2147483648), (2147483647, 2147483647),
                        (-2147483648, 2147483647), (2147483647, -2147483648)):
        add("/reservation", json.dumps(dict(now=now, expires_at=expiry)), 200, dict(active=now <= expiry), "i32-boundary")
    for unit, units, returned, shipping in ((2147483647, 1, 1, 0), (1073741823, 2, 1, 1),
                                             (0, 2147483647, 2147483647, 2147483647)):
        refund = unit * returned + (shipping if returned else 0)
        add("/refund", json.dumps(dict(unit_cents=unit, units=units, returned_units=returned, shipping_cents=shipping)),
            200, dict(refund_cents=refund, remaining_cents=unit * units + shipping - refund), "i32-boundary")
    # Both decoders must reject bytes that cannot be a UTF-8 JSON document.
    invalid_utf8 = json.dumps(checkout | {"sku": "INVALID"}).encode().replace(b"INVALID", b"\xff")
    add("/checkout", invalid_utf8, name="utf8-schema")
    return out


@contextlib.contextmanager
def service(executable, project):
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0)); address = f"127.0.0.1:{sock.getsockname()[1]}"
    with tempfile.TemporaryFile() as log:
        proc = subprocess.Popen([str(executable)], cwd=project, env=dict(os.environ, ADDR=address),
                                stdout=log, stderr=log, start_new_session=True)
        try:
            for _ in range(150):
                if proc.poll() is not None: raise RuntimeError("service exited before readiness")
                try:
                    with urllib.request.urlopen(f"http://{address}/health", timeout=.1) as response:
                        if response.status == 200: break
                except (OSError, urllib.error.URLError): time.sleep(.02)
            else: raise RuntimeError("service readiness timeout")
            yield f"http://{address}"
        finally:
            try: os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError: pass
            proc.wait()


def verify(base, case):
    body = case["body"]
    req = urllib.request.Request(base + case["path"], data=body if isinstance(body, bytes) else body.encode(),
                                 method=case["method"], headers={"Content-Type": "application/json"})
    try: response = urllib.request.urlopen(req, timeout=2)
    except urllib.error.HTTPError as error: response = error
    with response:
        return compare_response(case, response.status, response.read(65536).decode())


def run_parity(kiro, *, trusted_fixture=False):
    if not trusted_fixture:
        raise ValueError("explicit --trusted-fixture required; model candidate host execution is unsupported")
    projects = {"kiro": ROOT / "workload", "go": ROOT / "workload-go-v1"}
    selected = cases("baseline") + boundary_cases()
    report = dict(kind="deterministic-cross-language-fixture-validation-not-model-performance",
                  trusted_fixture=True, arithmetic_domain="signed i32 inputs and non-overflowing intermediate arithmetic",
                  cases_per_language=len(selected), languages={})
    with tempfile.TemporaryDirectory(prefix="kiro-go-parity-") as temp:
        for language, project in projects.items():
            executable = Path(temp) / language
            argv = [str(kiro), "build", ".", "--out", str(executable)] if language == "kiro" else ["go", "build", "-o", str(executable), "."]
            build = command(argv, project, 90)
            if build["exit_code"] or build["timed_out"]:
                report["languages"][language] = dict(passed=False, build=build); continue
            with service(executable, project) as base:
                results = [verify(base, case) for case in selected]
            report["languages"][language] = dict(passed=all(c["passed"] for c in results),
                                                   passed_cases=sum(c["passed"] for c in results),
                                                   failures=[c for c in results if not c["passed"]])
    report["passed"] = all(result["passed"] for result in report["languages"].values())
    return report


class ParityCasesTests(unittest.TestCase):
    def test_host_execution_requires_explicit_trust(self):
        with self.assertRaisesRegex(ValueError, "trusted-fixture"):
            run_parity("unused")
    def test_cli_saves_report_and_creates_parent_directories(self):
        report = dict(passed=True, kind="deterministic-fixture-check")
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp) / "nested" / "parity.json"
            with patch(__name__ + ".run_parity", return_value=report), contextlib.redirect_stdout(io.StringIO()) as stdout:
                result = main(["--trusted-fixture", "--kiro", "/unused", "--out", str(target)])
            self.assertEqual(result, 0)
            self.assertEqual(json.loads(target.read_text()), report)
            self.assertEqual(json.loads(stdout.getvalue()), report)

    def test_independent_cases_are_unique_and_cover_boundaries(self):
        selected = boundary_cases()
        self.assertEqual(len(selected), len({case["id"] for case in selected}))
        self.assertEqual({"schema", "i32-schema", "optional", "i32-boundary", "utf8-schema"}, {c["group"] for c in selected})
        self.assertTrue(any(isinstance(c["body"], bytes) for c in selected))


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--trusted-fixture", action="store_true")
    parser.add_argument("--kiro", type=Path, required=True)
    parser.add_argument("--out", type=Path, help="save reproducible validation report")
    args = parser.parse_args(argv)
    if not args.trusted_fixture: parser.error("--trusted-fixture is required")
    report = run_parity(args.kiro.resolve(), trusted_fixture=True)
    encoded = json.dumps(report, indent=2)
    if args.out is not None:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(encoded + "\n")
    print(encoded)
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
