"""Go harness regressions; full HTTP matrix: go_benchmark.py --selftest --trusted-fixture."""
import json
from pathlib import Path
import re
import shutil
import tempfile
import unittest
from unittest.mock import patch

import go_benchmark
import go_controls
from oracle import TASKS


class GoHarnessTests(unittest.TestCase):
    def test_payment_authorization_alias_preserves_all_nine_public_contracts(self):
        with tempfile.TemporaryDirectory() as tmp:
            project = Path(tmp) / "project"
            shutil.copytree(go_benchmark.ROOT / "workload-go-v1", project)
            go_controls.patch(project, "payment-boundary")
            checkout = project / "internal/checkout/checkout.go"
            text = re.sub(r"(?s)type Authorization struct \{.*?\}", "type Authorization = payments.Authorization", checkout.read_text(), count=1)
            checkout.write_text(text)
            report = go_benchmark.api_contracts(project)
            self.assertTrue(report["passed"], report)
            self.assertEqual(len(report["checks"]), 9)

    def test_host_execution_requires_explicit_attestation(self):
        with patch("go_benchmark.command") as command:
            with self.assertRaisesRegex(ValueError, "trusted-fixture"):
                go_benchmark.evaluate("/untrusted", "baseline")
            command.assert_not_called()

    def test_unused_payments_does_not_satisfy_boundary(self):
        graph = dict(nodes=[dict(id="payments.Unused", module="payments")], edges=[])
        self.assertFalse(go_benchmark.structure(graph, "payment-boundary")["passed"])

    def test_isolated_evaluation_requires_pinned_image_and_has_no_host_fallback(self):
        baseline = go_benchmark.ROOT / "workload-go-v1"
        with self.assertRaisesRegex(ValueError, "immutable"):
            go_benchmark.evaluate_isolated(baseline, "golang:latest", "baseline")
        calls = []
        def fail_create(argv, cwd, timeout):
            calls.append(argv)
            return dict(exit_code=1, timed_out=False, stderr="daemon unavailable", stdout="")
        with patch("go_benchmark.shared_command", fail_create), patch("go_benchmark.evaluate") as host:
            report = go_benchmark.evaluate_isolated(baseline, "sha256:" + "a"*64, "baseline")
            host.assert_not_called()
        self.assertEqual(report["status"], "blocked_environment")
        self.assertEqual(len(calls), 1)
        create = calls[0]
        self.assertEqual(create[create.index("--network")+1], "none")
        self.assertIn("--read-only", create)
        self.assertIn("no-new-privileges", create)
        mounts = [create[i+1] for i, arg in enumerate(create) if arg == "--mount"]
        self.assertEqual(len(mounts), 1)
        self.assertTrue(mounts[0].endswith("dst=/work,readonly"))
        self.assertFalse(any("oracle" in mount for mount in mounts))

    def test_isolated_evaluation_rejects_symlink_inputs(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); (root / "go.mod").symlink_to("/tmp/not-a-fixture")
            with self.assertRaisesRegex(ValueError, "symlinks"):
                go_benchmark.evaluate_isolated(root, "sha256:"+"a"*64, "baseline")

    def test_boundary_requires_linkage_ownership_and_independence(self):
        names = ("PaymentKnown", "PaymentFee", "PaymentReference")
        nodes = [dict(id="payments." + n, module="payments") for n in (*names, "Authorize", "Authorization", "Error")]
        next(n for n in nodes if n["id"] == "payments.Authorize")["returns"] = ["Authorization", "error"]
        edges = [dict(from_="checkout.Authorize", to="payments.Authorize")]
        edges += [dict(from_="payments.Authorize", to="payments." + n) for n in names]
        graph = dict(nodes=nodes, edges=[{"from": e["from_"], "to": e["to"]} for e in edges])
        self.assertTrue(go_benchmark.structure(graph, "payment-boundary")["passed"])
        graph["edges"].append({"from": "payments.__imports", "to": "pricing.__package"})
        self.assertFalse(go_benchmark.structure(graph, "payment-boundary")["passed"])
        graph["edges"].pop()
        graph["nodes"].append(dict(id="checkout.PaymentFee", module="checkout"))
        self.assertFalse(go_benchmark.structure(graph, "payment-boundary")["passed"])

    def test_ast_aliases_and_fake_comment_calls(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / "internal/checkout/checkout.go"
            source.parent.mkdir(parents=True)
            source.write_text('''package checkout
import pay "benchmark.local/checkout/internal/payments"
func Authorize() { pay.Authorize() }
// pay.PaymentFee() is a comment, not a dependency.
''')
            result = go_benchmark.command([go_benchmark.GO, "run", str(go_benchmark.HERE / "goinspect/main.go"), str(root)], go_benchmark.HERE)
            self.assertEqual(result["exit_code"], 0, result["stderr"])
            graph = json.loads(result["stdout"])
            calls = {(e["from"], e["to"]) for e in graph["edges"] if e["kind"] == "call"}
            self.assertIn(("checkout.Authorize", "payments.Authorize"), calls)
            self.assertNotIn(("checkout.Authorize", "payments.PaymentFee"), calls)

    def test_nested_module_import_identity_preserves_transitive_dependencies(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            sources = {
                "internal/payments/payments.go": '''package payments
import helper "benchmark.local/checkout/internal/neutral/helper"
type Authorization struct{}
func Authorize() (Authorization,error) {helper.Pass(); PaymentKnown(); PaymentFee(); PaymentReference(); return Authorization{},nil}
func PaymentKnown() {}
func PaymentFee() {}
func PaymentReference() {}
''',
                "internal/neutral/helper/helper.go": '''package helper
import "benchmark.local/checkout/internal/pricing"
func Pass() {pricing.Any()}
''',
                "internal/checkout/checkout.go": '''package checkout
import "benchmark.local/checkout/internal/payments"
func Authorize() {payments.Authorize()}
'''}
            for path, source in sources.items():
                dest = root / path; dest.parent.mkdir(parents=True, exist_ok=True); dest.write_text(source)
            result = go_benchmark.command([go_benchmark.GO, "run", str(go_benchmark.HERE / "goinspect/main.go"), str(root)], go_benchmark.HERE)
            self.assertEqual(result["exit_code"], 0, result["stderr"])
            graph = json.loads(result["stdout"])
            edges = {(e["from"],e["to"]) for e in graph["edges"]}
            self.assertIn(("payments.Authorize", "neutral/helper.Pass"), edges)
            self.assertIn(("neutral/helper.Pass", "pricing.Any"), edges)
            checks = go_benchmark.structure(graph, "payment-boundary")["checks"]
            self.assertEqual([c["name"] for c in checks if not c["passed"]], ["payments has no forbidden dependencies"])

    def test_all_controls_are_source_patches_on_fresh_baseline(self):
        baseline = go_benchmark.ROOT / "workload-go-v1"
        with tempfile.TemporaryDirectory() as tmp:
            for task in TASKS:
                for negative in (False, True):
                    project = Path(tmp) / f"{task}-{negative}"
                    shutil.copytree(baseline, project)
                    go_controls.patch(project, task, negative)
                    changed = [p for p in project.rglob("*.go") if not (baseline / p.relative_to(project)).exists() or p.read_bytes() != (baseline / p.relative_to(project)).read_bytes()]
                    self.assertTrue(changed, task)
                    result = go_benchmark.command([go_benchmark.GO, "test", "./..."], project)
                    self.assertEqual(result["exit_code"], 0, result["stdout"] + result["stderr"])


if __name__ == "__main__": unittest.main()
