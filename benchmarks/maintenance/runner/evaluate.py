#!/usr/bin/env python3
"""Evaluate a candidate through HTTP, independent of its own assertions.

Expected values stay on the host. Candidate execution stays in a network-disabled container.
Direct host execution is reserved for trusted repository fixtures during selftest.
"""
import argparse
import collections
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

from oracle import TASKS, cases, structure, api_contracts


def command(argv, cwd, timeout=60, max_output=131072):
    start = time.monotonic()
    with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
        proc = subprocess.Popen(argv, cwd=cwd, stdout=stdout, stderr=stderr, start_new_session=True)
        timed_out = False
        try:
            proc.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
        finally:
            # Kill remaining children too; exited parent does not imply no descendants.
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            proc.wait()
        stdout.seek(0); stderr.seek(0)
        out, err = stdout.read(max_output + 1), stderr.read(max_output + 1)
    return dict(argv=argv, exit_code=proc.returncode, timed_out=timed_out,
                seconds=time.monotonic() - start, stdout=out[:max_output].decode(errors="replace"),
                stderr=err[:max_output].decode(errors="replace"), truncated=len(out) > max_output or len(err) > max_output)


def request(base, case):
    req = urllib.request.Request(base + case["path"], data=case["body"].encode(), method=case["method"],
                                 headers={"Content-Type": "application/json"})
    try:
        response = urllib.request.urlopen(req, timeout=1)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        raw = response.read(65537)
        if len(raw) > 65536:
            raise ValueError("response exceeds limit")
        text = raw.decode("utf-8")
        return compare_response(case,response.status,text)


def compare_response(case,status,text):
    actual = json.loads(text) if isinstance(case["expected"], dict) else text
    # Python bools equal integers; compare exact JSON value types as well as values.
    equal = json.dumps(actual, sort_keys=True, separators=(",", ":")) == json.dumps(case["expected"], sort_keys=True, separators=(",", ":"))
    return dict(id=case["id"],group=case["group"],passed=status==case["status"] and (case["expected"] is None or equal),
                status=status,expected_status=case["status"],actual=actual,expected=case["expected"])


def finalize(report,task,start):
    counts = collections.defaultdict(lambda: dict(passed=0, total=0))
    for case in report["cases"]:
        group = counts[case["group"]]
        group["total"] += 1
        group["passed"] += int(case["passed"])
    report["groups"] = dict(counts)
    report["behavioral_passed"] = report["status"] == "complete" and len(report["cases"]) == len(cases(task)) and all(c["passed"] for c in report["cases"])
    report["passed"] = report["behavioral_passed"] and report["structural"]["passed"] and report.get("api",{}).get("passed",False)
    report["seconds"] = time.monotonic() - start
    return report



def evaluate(project, binary, task, timeout=120):
    start = time.monotonic()
    report = dict(schema="kiro.benchmark.evaluation.v1", task=task, passed=False,
                  status="verification_failed", steps=[], cases=[], structural=None,
                  kind="deterministic-evaluation-not-model-performance")
    for args in (("lint", ".", "--json"), ("test", ".")):
        step = command([binary, *args], project, min(60, timeout))
        report["steps"].append(step)
        if step["exit_code"] or step["timed_out"]:
            report["status"] = "timeout" if step["timed_out"] else "verification_failed"
            report["seconds"] = time.monotonic() - start
            return report
    step = command([binary,"symbols",".","--json"], project, 30, max_output=1048576)
    report["steps"].append(step)
    try:
        report["api"] = api_contracts(json.loads(step["stdout"])["data"]["symbols"])
    except (KeyError,ValueError):
        report["api"] = dict(passed=False,error="public API unavailable")
    with tempfile.TemporaryDirectory(prefix="kiro-benchmark-eval-") as tmp:
        executable = str(Path(tmp) / "service")
        step = command([binary, "build", ".", "--out", executable], project, min(60, timeout))
        report["steps"].append(step)
        if step["exit_code"] or step["timed_out"]:
            report["status"] = "timeout" if step["timed_out"] else "verification_failed"
            report["seconds"] = time.monotonic() - start
            return report
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            port = sock.getsockname()[1]
        env = dict(os.environ, ADDR=f"127.0.0.1:{port}")
        with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
            proc = subprocess.Popen([executable], cwd=project, env=env, stdout=stdout, stderr=stderr, start_new_session=True)
            base = f"http://127.0.0.1:{port}"
            try:
                ready = False
                for _ in range(100):
                    if proc.poll() is not None:
                        break
                    try:
                        with urllib.request.urlopen(base + "/health", timeout=0.1) as response:
                            response.read(65536)
                        ready = True
                        break
                    except (OSError, urllib.error.URLError):
                        time.sleep(0.02)
                if not ready:
                    report["status"] = "service_start_failed"
                else:
                    for case in cases(task):
                        if time.monotonic() - start > timeout:
                            report["status"] = "timeout"
                            break
                        try:
                            report["cases"].append(request(base, case))
                        except Exception as error:
                            report["cases"].append(dict(id=case["id"], group=case["group"], passed=False, error=str(error)))
                    else:
                        report["status"] = "complete"
            finally:
                try:
                    os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                proc.wait()
                stderr.seek(0)
                report["service_stderr"] = stderr.read(65536).decode(errors="replace")
    # Architecture scoring is evaluator work, independent of which tools the agent had.
    if task in ("payment-boundary", "payment-declined"):
        step = command([binary, "analyze", "graph", ".", "--json"], project, 30, max_output=8388608)
        report["steps"].append(step)
        if step["exit_code"] or step["timed_out"] or step["truncated"]:
            report["structural"] = dict(required=True, passed=False, error="graph unavailable or too large")
        else:
            try:
                report["structural"] = structure(json.loads(step["stdout"])["data"]["result"], task)
            except (KeyError, ValueError) as error:
                report["structural"] = dict(required=True, passed=False, error=str(error))
    else:
        report["structural"] = dict(required=False, passed=True, checks=[])
    return finalize(report,task,start)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--project", type=Path, required=True)
    parser.add_argument("--kiro", type=Path, required=True)
    parser.add_argument("--task", choices=(*TASKS, "baseline"), required=True)
    parser.add_argument("--timeout", type=float, default=120)
    args = parser.parse_args()
    print(json.dumps(evaluate(str(args.project.resolve()), str(args.kiro.resolve()), args.task, args.timeout)))
