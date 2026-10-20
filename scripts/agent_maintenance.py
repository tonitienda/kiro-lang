#!/usr/bin/env python3
"""Prepare quote-service maintenance tasks and evaluate candidates with fixed acceptance checks.

This runner does not invoke a model or claim a benchmark result. Agent effort/token
measurements must be recorded by the calling experiment. Evaluation copies the candidate
and adds acceptance tests kept outside its working directory.
"""
import argparse
import json
import shutil
import subprocess
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TASK_SCHEMA = "kiro.maintenance.v1"
TASKS = {
    "stock-error": {
        "requirement": "Treat SKU sold-out as a recognized but unavailable product. Add typed OutOfStock errors carrying sku in catalog and pricing, map explicitly at their boundary, and exhaustively map the pricing error to HTTP 422 with body 'out of stock: sold-out'. Preserve existing products, unknown-product behavior, routes and other pricing rules.",
        "cases": [
            ('{"sku":"sold-out","delivery":"standard"}', 422, "out of stock: sold-out"),
            ('{"sku":"missing","delivery":"standard"}', 422, "unknown product: missing"),
            ('{"sku":"book","delivery":"standard"}', 200, None),
        ],
        "domain": 'test.eq(actual:catalog.price(sku:"sold-out"),expected:Err(error:catalog.Error.OutOfStock(sku:"sold-out")))\ntest.eq(actual:pricing.quote(sku:"sold-out",addon:nil,delivery:"standard",coupon:nil),expected:Err(error:pricing.Error.OutOfStock(sku:"sold-out")))',
    },
    "delivery-schema": {
        "requirement": "Change only the HTTP request delivery field from a string to a closed record with method:str, e.g. delivery:{method:standard}. Use its method for existing pricing. String delivery and objects missing method must return 400. Preserve optional addon/coupon handling, health/method/routing behavior and pricing's existing public API. Update the project's request fixtures and tests for this intentional schema break.",
        "cases": [
            ('{"sku":"book","delivery":{"method":"standard"}}', 200, None),
            ('{"sku":"book","delivery":"standard"}', 400, None),
            ('{"sku":"book","delivery":{}}', 400, None),
            ('{"sku":"book","delivery":{"method":"teleport"}}', 422, "unknown delivery: teleport"),
        ],
        "domain": "",
    },
    "coupon-minimum": {
        "requirement": "SHIPFREE requires an item subtotal of at least 2000 cents. For smaller subtotals return pricing.Error.CouponMinimum(minimum_cents:i32,subtotal_cents:i32), mapped exhaustively to HTTP 422 with body 'coupon requires subtotal of 2000 cents'. Eligible quotes still waive shipping. Without the coupon, preserve current prices. Update fixtures/tests for this changed policy; keep other errors and routes intact.",
        "cases": [
            ('{"sku":"book","delivery":"standard","coupon":"SHIPFREE"}', 422, "coupon requires subtotal of 2000 cents"),
            ('{"sku":"book","addon":"poster","delivery":"express","coupon":"SHIPFREE"}', 200, None),
            ('{"sku":"book","delivery":"standard"}', 200, None),
        ],
        "domain": 'test.eq(actual:pricing.quote(sku:"book",addon:nil,delivery:"standard",coupon:"SHIPFREE"),expected:Err(error:pricing.Error.CouponMinimum(minimum_cents:2000,subtotal_cents:1500)))',
    },
}


def acceptance_source(task):
    # JSON quoting produces valid Go/Kiro quoted string spelling for this ASCII corpus.
    lines = ["mod maintenance_acceptance", "import app", "import catalog", "import pricing", "import http", "import json", "import test"]
    lines += ["type QuoteView {currency:str subtotal_cents:i32 shipping_cents:i32 total_cents:i32 coupon:?str}"]
    for index, (body, status, text) in enumerate(task["cases"]):
        lines += [f"fn test_acceptance_{index}()->R[nil,str] {{",
                  "let response = app.handler(req:http.test_req(method:\"POST\",path:\"/quotes\",body:" + json.dumps(body) + "))?",
                  f"test.eq(actual:response.code,expected:{status})"]
        if text is not None:
            lines += ["test.eq(actual:response.body,expected:" + json.dumps(text) + ")"]
        if status == 200:
            subtotal = 2200 if '"poster"' in body else 1500
            shipping = 0 if '"SHIPFREE"' in body else 300
            lines += ["let quote = json.decode[QuoteView](text:response.body)?",
                      f"test.eq(actual:quote.subtotal_cents,expected:{subtotal})",
                      f"test.eq(actual:quote.shipping_cents,expected:{shipping})",
                      f"test.eq(actual:quote.total_cents,expected:{subtotal + shipping})"]
        lines += ["return Ok(value:nil)", "}"]
    if task["domain"]:
        lines += ["fn test_domain_acceptance()->nil {", task["domain"], "return nil", "}"]
    lines += ['fn test_routes_acceptance()->R[nil,str] {',
              'let health = app.handler(req:http.test_req(method:"GET",path:"/health",body:""))?',
              'test.eq(actual:health.code,expected:200)',
              'test.eq(actual:health.body,expected:"ok")',
              'let missing = app.handler(req:http.test_req(method:"GET",path:"/unknown",body:""))?',
              'test.eq(actual:missing.code,expected:404)',
              'let wrong = app.handler(req:http.test_req(method:"GET",path:"/quotes",body:""))?',
              'test.eq(actual:wrong.code,expected:405)', 'return Ok(value:nil)', '}']
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    prepare = commands.add_parser("prepare")
    prepare.add_argument("--task", choices=TASKS, required=True)
    prepare.add_argument("--out", type=Path, required=True)
    evaluate = commands.add_parser("evaluate")
    evaluate.add_argument("--task", choices=TASKS, required=True)
    evaluate.add_argument("--candidate", type=Path, required=True)
    evaluate.add_argument("--kiro", type=Path, required=True)
    args = parser.parse_args()
    task = TASKS[args.task]
    if args.command == "prepare":
        if args.out.exists():
            parser.error("output already exists; choose a new task directory")
        args.out.mkdir(parents=True)
        shutil.copytree(ROOT / "examples/quote_service", args.out / "project")
        shutil.copytree(ROOT / "docs/llm", args.out / "project/.kiro/skill", ignore=shutil.ignore_patterns("*.go"))
        (args.out / "project/AGENTS.md").write_text("Read ../task.json and .kiro/skill/SKILL.md. Use kiro describe --json, kiro symbols . --json, kiro check . --json and kiro lint . --json. Implement and audit with agents and deterministic tools; no human source-review gate is assumed.\n")
        (args.out / "task.json").write_text(json.dumps({"schema":TASK_SCHEMA,"id": args.task, "requirement": task["requirement"],
            "baseline": "quote-service Phase 16 public behavior; copied sources are authoritative",
            "verification": "evaluate runs canonical-format/semantic checks, project tests and additional fixed acceptance cases"}, indent=2) + "\n")
        print(json.dumps({"task": args.task, "candidate": str((args.out / "project").resolve()), "instructions": str((args.out / "task.json").resolve())}))
        return
    binary = str(args.kiro.resolve())
    results = []
    with tempfile.TemporaryDirectory(prefix="kiro-maintenance-eval-") as temp:
        project = Path(temp) / "project"
        shutil.copytree(args.candidate, project)
        # Acceptance cases are added only in the evaluation copy. They are fixed in
        # the runner, not taken from agent-modified candidate tests or task.json.
        oracle = project / "maintenance_acceptance"
        oracle.mkdir(exist_ok=False)
        (oracle / "main.ki").write_text(acceptance_source(task))
        # Candidate formatting is checked independently; generated oracle spelling
        # is normalized separately without touching candidate sources.
        for command in [[binary, "lint", str(args.candidate.resolve()), "--json"],
                        [binary, "fmt", str(oracle)], [binary, "check", str(project), "--json"],
                        [binary, "test", str(project)]]:
            start = time.perf_counter()
            result = subprocess.run(command, text=True, capture_output=True)
            record = {"command": command[1], "exit_code": result.returncode,
                      "verification_seconds": time.perf_counter() - start,
                      "stdout": result.stdout, "stderr": result.stderr}
            if "--json" in command:
                record["report"] = json.loads(result.stdout)
            results.append(record)
            if result.returncode:
                break
    passed = len(results) == 4 and all(step["exit_code"] == 0 for step in results)
    print(json.dumps({"schema":TASK_SCHEMA,"kind": "maintenance-acceptance-not-agent-benchmark", "task": args.task,
        "passed": passed, "verification_tool_calls": len(results), "steps": results,
        "unmeasured": ["agent tokens", "agent tool calls", "repair attempts", "agent elapsed time"]}, indent=2))
    raise SystemExit(0 if passed else 1)


if __name__ == "__main__":
    main()
