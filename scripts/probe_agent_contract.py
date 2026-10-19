#!/usr/bin/env python3
"""Record checker/runtime disagreement; observations, not a conformance test suite.

Usage: python3 scripts/probe_agent_contract.py --kiro /absolute/path/to/kiro
"""

import argparse
import json
from pathlib import Path
import subprocess
import tempfile


PROBES = {
    "return_type": '''mod main

fn number() -> i32 {
  return "not an integer"
}

fn main() -> i32 !io {
  println(value : number())
  return 0
}
''',
    "json_required_field": '''mod main

import json

type Input {
  quantity:i32
}

fn main() -> i32 !io {
  let input = json.decode[Input](text : "{}")?
  println(value : json.encode(value : input)?)
  return 0
}
''',
    "documented_loop": '''mod main

fn main() -> i32 !io {
  let xs = [1, 2, 3]
  mut sum = 0
  for x in xs {
    sum = sum + x
  }
  println(value : sum)
  return 0
}
''',
    "result_propagation": '''mod main

import test

fn leaf() -> R[i32,str] {
  return Err(error : "expected domain error")
}

fn parent() -> R[i32,str] {
  let value = leaf()?
  return Ok(value : value)
}

fn main() -> i32 {
  test.eq(actual : parent(), expected : Err(error : "expected domain error"))
  return 0
}
''',
}


def invoke(command, cwd):
    try:
        result = subprocess.run(command, cwd=cwd, text=True, capture_output=True, timeout=60)
        return {"exit_code": result.returncode, "stdout": result.stdout, "stderr": result.stderr}
    except subprocess.TimeoutExpired:
        return {"exit_code": None, "stdout": "", "stderr": "probe timed out after 60s"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--kiro", required=True)
    args = parser.parse_args()
    kiro = str(Path(args.kiro).resolve(strict=True))
    records = []
    with tempfile.TemporaryDirectory(prefix="kiro-contract-") as root:
        for name, source in PROBES.items():
            directory = Path(root) / name
            directory.mkdir()
            (directory / "main.ki").write_text(source, encoding="utf-8")
            check = invoke([kiro, "check", "."], directory)
            run = invoke([kiro, "run", "."], directory)
            records.append({"probe": name, "check": check, "run": run})
    print(json.dumps({"kind": "observations-not-agent-benchmark", "probes": records}, indent=2))


if __name__ == "__main__":
    main()
