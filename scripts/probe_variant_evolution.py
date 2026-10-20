#!/usr/bin/env python3
"""Assert that adding an error variant invalidates the affected quote-service handler.

This tests compiler diagnostics, not agent repair performance. Source is copied into
an isolated temporary directory; the repository example is never modified.
"""
import argparse
import json
import shutil
import subprocess
import tempfile
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--kiro", required=True)
    args = parser.parse_args()
    binary = str(Path(args.kiro).resolve())
    example = Path(__file__).resolve().parents[1] / "examples/quote_service"
    mutations = [
        ("internal/catalog/main.ki", "OutOfStock", "pricing.from_catalog"),
        ("internal/shipping/main.ki", "Timeout", "pricing.from_shipping"),
        ("internal/pricing/main.ki", "PriceOverflow", "app.invalid_quote"),
    ]
    observations = []
    for relative, tag, handler in mutations:
        with tempfile.TemporaryDirectory(prefix="kiro-variant-evolution-") as temp:
            project = Path(temp) / "quote"
            shutil.copytree(example, project)
            baseline = subprocess.run([binary, "check", str(project), "--json"], text=True, capture_output=True)
            if baseline.returncode:
                raise RuntimeError("baseline failed: " + baseline.stderr)
            source = project / relative
            text = source.read_text()
            marker = "type Error {"
            if text.count(marker) != 1:
                raise RuntimeError("expected one Error declaration in " + relative)
            source.write_text(text.replace(marker, marker + "\n  variant " + tag + " {}", 1))
            result = subprocess.run([binary, "check", str(project), "--json"], text=True, capture_output=True)
            report = json.loads(result.stdout)
            diagnostic = report["diagnostics"][0] if report["diagnostics"] else {}
            if (result.returncode == 0 or result.stderr or diagnostic.get("code") != "KIRO_NON_EXHAUSTIVE"
                    or diagnostic.get("symbol") != handler or tag not in diagnostic.get("missing", [])):
                raise RuntimeError("mutation did not produce expected structured diagnostic: " + result.stdout + result.stderr)
            observations.append({"added_variant": tag, "handler": handler, "exit_code": result.returncode,
                                 "code": diagnostic["code"], "file": diagnostic.get("file"),
                                 "location_kind": diagnostic.get("location_kind"), "missing": diagnostic["missing"]})
    print(json.dumps({"kind": "compiler-mutation-check-not-agent-benchmark", "observations": observations}, indent=2))


if __name__ == "__main__":
    main()
