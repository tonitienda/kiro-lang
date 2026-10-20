#!/usr/bin/env python3
"""Deterministic Go fixture evaluator. This is NOT an arbitrary/model-code sandbox.

Host execution requires explicit attestation that the project is a trusted fixture.
Arbitrary candidate evaluation requires --image with a pinned Docker image; no
host fallback is permitted when isolated evaluation is unavailable.
"""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import signal
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

from evaluate import command as shared_command, compare_response, finalize, request
from oracle import TASKS, cases

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent
GO = shutil.which("go") or "go"


def command(argv, cwd, timeout=60):
    if argv[0] == GO:
        cache = Path(tempfile.gettempdir()) / "kiro-maintenance-go-cache"
        cache.mkdir(exist_ok=True)
        argv = ["/usr/bin/env", "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", f"GOCACHE={cache}", *argv]
    return shared_command(argv, cwd, timeout)


def validate_module(project):
    module_file = Path(project) / "go.mod"
    if not module_file.is_file() or not re.search(r"(?m)^module\s+benchmark\.local/checkout\s*$", module_file.read_text()):
        raise ValueError("Go fixture module must remain benchmark.local/checkout")


def structure(graph, task):
    if task not in ("payment-boundary", "payment-declined"):
        return dict(required=False, passed=True, checks=[])
    nodes = {n["id"] for n in graph["nodes"]}
    edges = {(e["from"], e["to"]) for e in graph["edges"]}
    checks = []
    def check(name, value): checks.append(dict(name=name, passed=bool(value)))
    if task == "payment-declined":
        outcomes = {"checkout.ErrPaymentDeclined", "checkout.PaymentDeclined"} & nodes
        check("distinct checkout declined outcome", outcomes)
        check("authorization constructs declined outcome", any(("checkout.Authorize", outcome) in edges for outcome in outcomes))
    else:
        check("payments owns authorization type", "payments.Authorization" in nodes)
        check("payments authorization owns policy", "payments.Authorize" in nodes)
        result_types = next((n.get("returns", []) for n in graph["nodes"] if n["id"] == "payments.Authorize"), [])
        check("payments authorization returns its record and ordinary error", result_types in (["Authorization", "error"], ["Authorization", "Error"]))
        check("compatibility adapter calls payment module", ("checkout.Authorize", "payments.Authorize") in edges)
        for name in ("PaymentKnown", "PaymentFee", "PaymentReference"):
            names = {name, name[0].lower() + name[1:]}
            check(f"payments owns and calls {name}", any("payments." + n in nodes and ("payments.Authorize", "payments." + n) in edges for n in names)
                  and not any("checkout." + n in nodes for n in names))
        forbidden = {"checkout", "inventory", "pricing", "shipping", "catalog", "app", "reservations", "refunds"}
        dependencies = {}
        for a, b in edges: dependencies.setdefault(a.split(".")[0], set()).add(b.split(".")[0])
        reachable = set(); pending = ["payments"]
        while pending:
            module = pending.pop()
            if module in reachable: continue
            reachable.add(module); pending.extend(dependencies.get(module, set()) - reachable)
        check("payments has no forbidden dependencies", not (reachable & forbidden))
    return dict(required=True, passed=all(c["passed"] for c in checks), checks=checks)


# Compile assignments check actual public Go types, not source spelling or HTTP output.
# Probes live in package-local tests and check all nine preserved public APIs.
PROBES = {
 "app.Handler": ("app", 'import "net/http"\nvar _ func(http.ResponseWriter, *http.Request) = Handler'),
 "checkout.Authorization": ("internal/checkout", 'var _ = Authorization{Method: string(""), FeeCents: int32(0), Reference: string("")}\nfunc contractAuthorization(a Authorization) {var _ string = a.Method; var _ int32 = a.FeeCents; var _ string = a.Reference}'),
 "checkout.Receipt": ("internal/checkout", 'var _ = Receipt{SKU: string(""), Units: int32(0), RemainingStock: int32(0), SubtotalCents: int32(0), DiscountCents: int32(0), ShippingCents: int32(0), FeeCents: int32(0), TotalCents: int32(0), PaymentReference: string("")}'),
 "checkout.Authorize": ("internal/checkout", 'var _ func(string, int32) (Authorization, error) = Authorize'),
 "checkout.Place": ("internal/checkout", 'var _ func(string, int32, int32, string, *string, string) (Receipt, error) = Place'),
 "inventory.Remaining": ("internal/inventory", 'var _ func(int32, int32) (int32, error) = Remaining'),
 "pricing.QuoteFor": ("internal/pricing", 'var _ func(string, int32, string, *string) (Quote, error) = QuoteFor\nvar _ = Quote{SubtotalCents: int32(0), DiscountCents: int32(0), ShippingCents: int32(0), AmountCents: int32(0)}'),
 "refunds.Calculate": ("internal/refunds", 'var _ func(int32, int32, int32, int32) (Refund, error) = Calculate\nvar _ = Refund{RefundCents: int32(0), RemainingCents: int32(0)}'),
 "reservations.StatusAt": ("internal/reservations", 'var _ func(int32, int32) Status = StatusAt\nvar _ = Status{Active: bool(false)}'),
}

# Read fields back into exact types as well: assigning a typed literal alone
# would accidentally accept a field weakened to interface{}.
for _symbol, _type, _fields in (
    ("checkout.Authorization", "Authorization", {"Method":"string", "FeeCents":"int32", "Reference":"string"}),
    ("checkout.Receipt", "Receipt", {"SKU":"string", "Units":"int32", "RemainingStock":"int32", "SubtotalCents":"int32", "DiscountCents":"int32", "ShippingCents":"int32", "FeeCents":"int32", "TotalCents":"int32", "PaymentReference":"string"}),
    ("pricing.QuoteFor", "Quote", {"SubtotalCents":"int32", "DiscountCents":"int32", "ShippingCents":"int32", "AmountCents":"int32"}),
    ("refunds.Calculate", "Refund", {"RefundCents":"int32", "RemainingCents":"int32"}),
    ("reservations.StatusAt", "Status", {"Active":"bool"}),
):
    _package, _source = PROBES[_symbol]
    _readback = "; ".join(f"var _ {typ} = value.{name}" for name, typ in _fields.items())
    PROBES[_symbol] = (_package, _source + f"\nfunc benchmarkRead{_type}(value {_type}) {{{_readback}}}\n")
_package, _source = PROBES["checkout.Authorization"]
PROBES["checkout.Authorization"] = (_package, _source + "\nvar _ Error = (error)(nil)\nvar _ error = (Error)(nil)\n")


def api_contracts(project):
    with tempfile.TemporaryDirectory(prefix="go-api-") as tmp:
        copy = Path(tmp) / "project"
        shutil.copytree(project, copy, ignore=shutil.ignore_patterns(".git"))
        for index, (symbol, (package, source)) in enumerate(PROBES.items()):
            directory = copy / package
            directory.mkdir(parents=True, exist_ok=True)
            probe = directory / f"zz_benchmark_contract_{index}_test.go"
            if probe.exists(): raise ValueError("fixture collides with reserved API probe")
            probe.write_text(f'package {package.split("/")[-1]}\n{source}\n')
        step = command([GO, "test", "-run", "^$", "./..."], copy, 60)
        passed = step["exit_code"] == 0 and not step["timed_out"]
        return dict(passed=passed, checks=[dict(symbol=symbol, passed=passed) for symbol in PROBES],
                    stderr=step["stderr"], stdout=step["stdout"], probe_compilation="all-nine-contracts-together")


def evaluate_isolated(project, image, task, timeout=150):
    """Run arbitrary candidate code exclusively in a pinned, network-disabled image.

    Image must supply /opt/go/bin/go, python3, and sleep. Oracle expectations
    remain in this host process; only request bodies enter the container.
    """
    from broker import DockerSandbox
    from session import Driver, HTTP_TRANSPORT
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", image):
        raise ValueError("--image must be an immutable sha256 image ID")
    start = time.monotonic(); deadline = start + timeout
    project = Path(project).resolve()
    report = dict(schema="kiro.benchmark.evaluation.v1", language="go", task=task, passed=False,
                  status="verification_failed", steps=[], cases=[], structural=dict(required=False, passed=True, checks=[]),
                  kind="deterministic-evaluation-not-model-performance", isolation="docker-network-none", image_id=image)
    # Reject symlinks before copying: candidate input must not expose host files.
    if any(p.is_symlink() for p in project.rglob("*")):
        raise ValueError("candidate workspace contains symlinks")
    validate_module(project)
    if sum(p.stat().st_size for p in project.rglob("*") if p.is_file()) > 8 * 1024 * 1024:
        raise ValueError("candidate workspace exceeds size bound")
    class Sandbox(DockerSandbox):
        def __enter__(self):
            argv = ["docker", "create", "--name", self.name, "--network", "none", "--read-only",
                    "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "256",
                    "--memory", "1536m", "--cpus", "1", "--user", f"{os.getuid()}:{os.getgid()}",
                    "--mount", f"type=bind,src={self.project},dst=/work,readonly",
                    "--tmpfs", "/tmp:rw,exec,nosuid,size=768m,mode=1777",
                    "--tmpfs", "/cache:rw,nosuid,size=384m,mode=1777",
                    "--env", "HOME=/tmp", "--env", "GOCACHE=/cache/go", "--env", "GOMODCACHE=/cache/mod",
                    "--env", "CGO_ENABLED=0", "--env", "GOTOOLCHAIN=local", "--env", "GOENV=off",
                    "--env", "GOWORK=off", "--env", "GOPROXY=off", "--env", "GOSUMDB=off",
                    "--workdir", "/work", self.image, "sleep", "infinity"]
            result = shared_command(argv, self.project, 30)
            if result["exit_code"] or result["timed_out"]:
                raise RuntimeError("isolated Go container creation failed: " + result["stderr"])
            result = shared_command(["docker", "start", self.name], self.project, 30)
            if result["exit_code"] or result["timed_out"]:
                self.close(); raise RuntimeError("isolated Go container start failed: " + result["stderr"])
            return self
        def go(self, args):
            step = shared_command(["docker", "exec", self.name, "/opt/go/bin/go", *args], self.project, min(60, max(.01, deadline-time.monotonic())))
            if step["timed_out"]:
                self.close(); raise TimeoutError("Go command timed out; container terminated")
            return step
    try:
        with tempfile.TemporaryDirectory(prefix="go-isolated-eval-") as tmp:
            copy = Path(tmp) / "project"
            shutil.copytree(project, copy, ignore=shutil.ignore_patterns(".git"))
            # A single compilation includes all nine host-authored API probes.
            for index, (_, (package, source)) in enumerate(PROBES.items()):
                directory = copy / package
                directory.mkdir(parents=True, exist_ok=True)
                probe = directory / f"zz_benchmark_contract_{index}_test.go"
                if probe.exists(): raise ValueError("candidate collides with reserved API probe")
                probe.write_text(f'package {package.split("/")[-1]}\n{source}\n')
            with Sandbox(image, copy) as sandbox:
                target = sandbox.go(["env", "GOOS", "GOARCH"])
                if target["exit_code"] or len(target["stdout"].split()) != 2:
                    raise RuntimeError("isolated Go target architecture unavailable")
                target_platform = target["stdout"].split()
                for args in (["vet", "./..."], ["test", "./..."]):
                    step = sandbox.go(args); report["steps"].append(step)
                    if step["exit_code"]:
                        report["api"] = dict(passed=False, checks=[dict(symbol=s, passed=False) for s in PROBES])
                        return finalize(report, task, start)
                report["api"] = dict(passed=True, checks=[dict(symbol=s, passed=True) for s in PROBES])
                step = sandbox.go(["build", "-o", "/tmp/service", "."]); report["steps"].append(step)
                if step["exit_code"]: return finalize(report, task, start)
                result = shared_command(["docker", "exec", "-d", "--env", "ADDR=127.0.0.1:8080", sandbox.name, "/tmp/service"], copy, 10)
                if result["exit_code"]:
                    report["status"] = "service_start_failed"; return finalize(report, task, start)
                with tempfile.TemporaryFile() as stderr:
                    transport = Driver(["docker", "exec", "-i", sandbox.name, "python3", "-u", "-c", HTTP_TRANSPORT], stderr, 3)
                    try:
                        for _ in range(30):
                            reply = transport.exchange(dict(path="/health", method="GET", body=""), max(.01, deadline-time.monotonic()))
                            if reply.get("status") == 200: break
                            time.sleep(.02)
                        else:
                            report["status"] = "service_start_failed"; return finalize(report, task, start)
                        for case in cases(task):
                            if time.monotonic() >= deadline:
                                report["status"] = "timeout"; break
                            reply = transport.exchange({key:case[key] for key in ("path", "method", "body")}, deadline-time.monotonic())
                            try:
                                if "error" in reply: raise ValueError(reply["error"])
                                report["cases"].append(compare_response(case, reply["status"], reply["body"]))
                            except (ValueError, KeyError) as error:
                                report["cases"].append(dict(id=case["id"], group=case["group"], passed=False, error=str(error)))
                        else: report["status"] = "complete"
                    finally: transport.close()
            # Parse source with trusted helper only; no candidate compiler/plugin on host.
            step = command([GO, "run", str(HERE / "goinspect/main.go"), str(copy), *target_platform], HERE, 30)
            if step["exit_code"] or step["timed_out"] or step["truncated"]:
                report["structural"] = dict(required=True, passed=False, error="Go AST graph unavailable")
            else: report["structural"] = structure(json.loads(step["stdout"]), task)
    except (OSError, RuntimeError, TimeoutError, ValueError) as error:
        report["status"] = "blocked_environment" if isinstance(error, (OSError, RuntimeError)) else "verification_failed"
        report["error"] = str(error)
    return finalize(report, task, start)


def evaluate(project, task, timeout=120, *, trusted_fixture=False):
    if not trusted_fixture:
        raise ValueError("Go host execution requires explicit --trusted-fixture; arbitrary model trial evaluation is unsupported")
    project = Path(project).resolve()
    validate_module(project)
    start = time.monotonic()
    report = dict(schema="kiro.benchmark.evaluation.v1", language="go", task=task, passed=False,
                  status="verification_failed", steps=[], cases=[], structural=dict(required=False, passed=True, checks=[]),
                  kind="deterministic-evaluation-not-model-performance", trusted_fixture=True)
    for argv in ([GO, "vet", "./..."], [GO, "test", "./..."]):
        step = command(argv, project, min(60, timeout)); report["steps"].append(step)
        if step["exit_code"] or step["timed_out"]:
            report["status"] = "timeout" if step["timed_out"] else "verification_failed"
            return finalize(report, task, start)
    report["api"] = api_contracts(project)
    step = command([GO, "run", str(HERE / "goinspect/main.go"), str(project)], HERE, 30)
    report["steps"].append(step)
    try:
        if step["exit_code"] or step["timed_out"] or step["truncated"]: raise ValueError("Go AST graph unavailable")
        report["structural"] = structure(json.loads(step["stdout"]), task)
    except (ValueError, KeyError) as error:
        report["structural"] = dict(required=True, passed=False, error=str(error))
    with tempfile.TemporaryDirectory(prefix="go-benchmark-eval-") as tmp:
        executable = str(Path(tmp) / "service")
        step = command([GO, "build", "-o", executable, "."], project, min(60, timeout)); report["steps"].append(step)
        if step["exit_code"] or step["timed_out"]: return finalize(report, task, start)
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0)); port = sock.getsockname()[1]
        with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
            proc = subprocess.Popen([executable], cwd=project, env=dict(os.environ, ADDR=f"127.0.0.1:{port}"), stdout=stdout, stderr=stderr, start_new_session=True)
            base = f"http://127.0.0.1:{port}"
            try:
                for _ in range(100):
                    if proc.poll() is not None: break
                    try:
                        with urllib.request.urlopen(base + "/health", timeout=.1) as response: response.read(65536)
                        break
                    except (OSError, urllib.error.URLError): time.sleep(.02)
                else: pass
                try:
                    with urllib.request.urlopen(base + "/health", timeout=.1) as response: response.read(65536)
                except (OSError, urllib.error.URLError):
                    report["status"] = "service_start_failed"
                    return finalize(report, task, start)
                for case in cases(task):
                    if time.monotonic() - start > timeout:
                        report["status"] = "timeout"; break
                    try: report["cases"].append(request(base, case))
                    except Exception as error: report["cases"].append(dict(id=case["id"], group=case["group"], passed=False, error=str(error)))
                else: report["status"] = "complete"
            finally:
                try: os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError: pass
                proc.wait(); stderr.seek(0); report["service_stderr"] = stderr.read(65536).decode(errors="replace")
    return finalize(report, task, start)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--project", type=Path)
    parser.add_argument("--task", choices=(*TASKS, "baseline"))
    parser.add_argument("--selftest", action="store_true", help="validate trusted baseline and all reference/negative controls")
    parser.add_argument("--timeout", type=float, default=120)
    parser.add_argument("--trusted-fixture", action="store_true", help="attest this is trusted fixture code, safe to execute on host")
    parser.add_argument("--image", help="immutable Docker image ID for isolated candidate evaluation")
    parser.add_argument("--out", type=Path, help="write JSON report to this file")
    args = parser.parse_args()
    if args.trusted_fixture and args.image: parser.error("choose --image or --trusted-fixture")
    if not args.trusted_fixture and not args.image: parser.error("--image or --trusted-fixture is required; arbitrary model host evaluation is unsupported")
    if args.selftest:
        if not args.trusted_fixture: parser.error("selftest requires --trusted-fixture")
        report = selftest(args.timeout)
    else:
        if not args.project or not args.task: parser.error("--project and --task are required without --selftest")
        report = evaluate_isolated(args.project, args.image, args.task, args.timeout) if args.image else evaluate(args.project, args.task, args.timeout, trusted_fixture=True)
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(report, indent=2) + "\n")
    else: print(json.dumps(report))
    if args.selftest and not report["passed"]: raise SystemExit(1)


def selftest(timeout=120):
    from go_controls import patch
    reports = []
    baseline = ROOT / "workload-go-v1"
    reports.append(dict(control="baseline", expected_passed=True, evaluation=evaluate(baseline, "baseline", timeout, trusted_fixture=True)))
    for task in TASKS:
        reports.append(dict(control="unchanged", expected_passed=False, evaluation=evaluate(baseline, task, timeout, trusted_fixture=True)))
    with tempfile.TemporaryDirectory(prefix="go-controls-") as tmp:
        for task in TASKS:
            for negative in (False, True):
                project = Path(tmp) / f"{task}-{'negative' if negative else 'reference'}"
                shutil.copytree(baseline, project)
                patch(project, task, negative)
                reports.append(dict(control="negative" if negative else "reference", expected_passed=not negative,
                                    evaluation=evaluate(project, task, timeout, trusted_fixture=True)))
    return dict(schema="kiro.benchmark.go-selftest.v1", kind="deterministic-evaluation-not-model-performance",
                language="go", model_trials_run=0, passed=all(r["evaluation"]["passed"] == r["expected_passed"] for r in reports), reports=reports)


if __name__ == "__main__": main()
