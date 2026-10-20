"""Independent regressions for Go package ownership and transitive dependencies."""
import json
from pathlib import Path
import tempfile
import unittest

import go_benchmark


def valid_boundary_graph():
    names = ("Authorization", "Authorize", "PaymentKnown", "PaymentFee", "PaymentReference")
    return dict(nodes=[dict(id="payments." + name, module="payments", **({"returns": ["Authorization", "error"]} if name == "Authorize" else {})) for name in names],
                edges=[{"from": "checkout.Authorize", "to": "payments.Authorize", "kind": "call"}] +
                      [{"from": "payments.Authorize", "to": "payments." + name, "kind": "call"}
                       for name in names[2:]])


class IndependentGoStructureTests(unittest.TestCase):
    def test_transitive_forbidden_dependency_through_neutral_helper_is_rejected(self):
        graph = valid_boundary_graph()
        self.assertTrue(go_benchmark.structure(graph, "payment-boundary")["passed"])
        graph["edges"] += [dict(**{"from": "payments.__imports", "to": "neutral.__package"}, kind="dependency"),
                           dict(**{"from": "neutral.__imports", "to": "pricing.__package"}, kind="dependency")]
        report = go_benchmark.structure(graph, "payment-boundary")
        self.assertFalse(report["passed"])
        self.assertFalse(next(check["passed"] for check in report["checks"] if check["name"] == "payments has no forbidden dependencies"))
    def test_owned_authorization_required_without_nominal_error_boilerplate(self):
        graph = valid_boundary_graph()
        self.assertTrue(go_benchmark.structure(graph, "payment-boundary")["passed"])
        graph["nodes"] = [node for node in graph["nodes"] if node["id"] != "payments.Authorization"]
        self.assertFalse(go_benchmark.structure(graph, "payment-boundary")["passed"])
    def test_declined_local_name_cannot_fake_package_outcome_reference(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            directory = root / "internal/checkout"
            directory.mkdir(parents=True)
            (directory / "main.go").write_text("""package checkout
var ErrPaymentDeclined error
func Authorize() error { ErrPaymentDeclined := error(nil); return ErrPaymentDeclined }
func Actual() error { return ErrPaymentDeclined }
""")
            result = go_benchmark.command([go_benchmark.GO, "run", str(go_benchmark.HERE / "goinspect/main.go"), str(root)], go_benchmark.HERE)
            self.assertEqual(result["exit_code"], 0, result["stderr"])
            graph = json.loads(result["stdout"])
            references = {(edge["from"], edge["to"]) for edge in graph["edges"] if edge["kind"] == "reference"}
            self.assertNotIn(("checkout.Authorize", "checkout.ErrPaymentDeclined"), references)
            self.assertIn(("checkout.Actual", "checkout.ErrPaymentDeclined"), references)
            self.assertFalse(go_benchmark.structure(graph, "payment-declined")["passed"])

    def test_indirect_import_graph_and_shadowed_alias_calls(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for module, source in {
                "payments": '''package payments
import helper "benchmark.local/checkout/internal/neutral"
func Authorize() { helper.Work() }
func Shadow() { helper := struct{ Work func() }{}; helper.Work() }
''',
                "neutral": '''package neutral
import "benchmark.local/checkout/internal/pricing"
func Work() { pricing.Price() }
''',
            }.items():
                directory = root / "internal" / module
                directory.mkdir(parents=True)
                (directory / "main.go").write_text(source)
            result = go_benchmark.command([go_benchmark.GO, "run", str(go_benchmark.HERE / "goinspect/main.go"), str(root)], go_benchmark.HERE)
            self.assertEqual(result["exit_code"], 0, result["stderr"])
            graph = json.loads(result["stdout"])
            calls = {(edge["from"], edge["to"]) for edge in graph["edges"] if edge["kind"] == "call"}
            self.assertIn(("payments.Authorize", "neutral.Work"), calls)
            self.assertNotIn(("payments.Shadow", "neutral.Work"), calls)
            report = go_benchmark.structure(graph, "payment-boundary")
            self.assertFalse(next(check["passed"] for check in report["checks"] if check["name"] == "payments has no forbidden dependencies"))


if __name__ == "__main__": unittest.main()
