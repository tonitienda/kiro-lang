#!/usr/bin/env python3
"""Prepare, validate and run paired maintenance trials through an external model driver."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import random
import shutil
import subprocess
import sys
import tempfile
import time

from broker import Broker, doctor
from controls import patch
from evaluate import command, evaluate
from oracle import TASKS
from session import evaluate_isolated, run_session

ROOT = Path(__file__).resolve().parents[3]
SUITE = ROOT / "benchmarks/maintenance"
SCHEMA = "kiro.benchmark.plan.v1"


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def files_hash(root):
    return {str(p.relative_to(root)): sha(p) for p in sorted(Path(root).rglob("*"))
            if p.is_file() and "__pycache__" not in p.parts and p.suffix != ".pyc"}


def source_identity():
    paths = [SUITE / "workload", SUITE / "tasks", SUITE / "runner", ROOT / "docs/llm"]
    return {str(path.relative_to(ROOT)): files_hash(path) for path in paths}


def prepare(args):
    if args.out.exists():
        raise ValueError("output already exists")
    binary = str(args.kiro.resolve())
    discover = json.loads(subprocess.check_output([binary, "describe", "--json"], text=True))
    reference = json.loads(subprocess.check_output([binary, "describe", "--reference", "--json"], text=True))
    if hashlib.sha256(reference["data"]["content"].encode()).hexdigest() != reference["data"]["language_reference"]["sha256"]:
        raise ValueError("reference hash mismatch")
    if discover["data"]["language_reference"] != reference["data"]["language_reference"]:
        raise ValueError("discovery/reference mismatch")
    local_manifest=json.loads((ROOT/"docs/llm/references/kiro.json").read_text())
    if sha(ROOT/"docs/llm/references/language.md")!=reference["data"]["language_reference"]["sha256"] or local_manifest["version"]!=reference["data"]["guide_version"]:
        raise ValueError("vendored guide does not match compiler; rebuild before preparing cohort")
    schedule = []
    for repeat in range(args.repetitions):
        for task in TASKS:
            pair = ["core", "analysis"]
            random.Random(args.seed + repeat * 101 + TASKS.index(task)).shuffle(pair)
            for condition in pair:
                schedule.append(dict(id=f"r{repeat+1}-{task}-{condition}", repeat=repeat+1, task=task, condition=condition))
    # Randomize task blocks; keep each condition pair adjacent to reduce temporal drift.
    pairs = [schedule[i:i+2] for i in range(0,len(schedule),2)]
    random.Random(args.seed).shuffle(pairs)
    schedule = [trial for pair in pairs for trial in pair]
    plan = dict(schema=SCHEMA, kind="prepared-not-executed", repetitions=args.repetitions,
                seed=args.seed, source_commit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
                source_dirty=bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT)),
                compiler_sha256=sha(binary), guide=reference["data"]["language_reference"],
                guide_version=reference["data"]["guide_version"], inputs=source_identity(),
                limits=dict(tool_calls=120, trial_seconds=900, response_seconds=120, tool_seconds=60),
                schedule=schedule, model=None, model_settings=None,
                metrics=dict(primary="behavioral correctness and explicit structural contract", usage="driver reported or unavailable"))
    args.out.mkdir(parents=True)
    (args.out / "plan.json").write_text(json.dumps(plan,indent=2)+"\n")
    (args.out / "describe.json").write_text(json.dumps(discover,indent=2)+"\n")
    (args.out / "reference.json").write_text(json.dumps(reference,indent=2)+"\n")
    print(json.dumps(dict(plan=str(args.out / "plan.json"), trials=len(schedule), model_trials_run=0)))


def make_workspace(dest, task, condition):
    shutil.copytree(SUITE / "workload", dest)
    shutil.copytree(ROOT / "docs/llm", dest / ".kiro/skill", ignore=shutil.ignore_patterns("*.go", "__pycache__"))
    task_data = json.loads((SUITE / "tasks" / f"{task}.json").read_text())
    (dest / "task.json").write_text(json.dumps(task_data,indent=2)+"\n")
    allowed = "describe, symbols, fmt, check, lint and test" + (", plus optional analyze" if condition=="analysis" else "; analyze is unavailable")
    (dest / "AGENTS.md").write_text(f'''# Kiro benchmark task
This is a fresh maintenance trial. Do not assume Kiro syntax from another language.
Before editing, read task.json, .kiro/skill/SKILL.md and .kiro/skill/references/kiro.json.
Use .kiro/skill/references/language.md as the primary language reference, reading relevant
sections. Use kiro describe --json and kiro symbols . --json for capabilities/contracts.
Allowed Kiro commands: {allowed}.
Implement, audit and verify with the supplied tools; no human source-review gate is assumed.
Use the tool protocol only: no host shell, network, other agents or out-of-workspace access.
Do not alter pinned task/guide/instruction files. Add meaningful tests for the requested change.
Finish when done; independent acceptance is run after the session and is not tool-accessible.
''')
    return task_data


def selftest(args):
    binary = str(args.kiro.resolve())
    started = time.monotonic()
    results = []
    baseline = evaluate(str(SUITE / "workload"), binary, "baseline")
    results.append(dict(task="baseline", variant="unchanged", expected=True, passed=baseline["passed"], evaluation=baseline))
    with tempfile.TemporaryDirectory(prefix="kiro-benchmark-controls-") as tmp:
        for task in TASKS:
            for variant in ("unchanged", "reference", "negative"):
                project = Path(tmp) / f"{task}-{variant}"
                shutil.copytree(SUITE / "workload", project)
                if variant != "unchanged":
                    patch(project,task,negative=variant=="negative")
                formatted = command([binary,"fmt",str(project)],str(ROOT),30)
                if formatted["exit_code"]:
                    result = dict(passed=False,status="fixture_format_error",steps=[formatted])
                else:
                    result = evaluate(str(project),binary,task)
                results.append(dict(task=task,variant=variant,expected=variant=="reference",passed=result["passed"],evaluation=result))
                print(f"{task} {variant}: {'pass' if result['passed'] else 'rejected'}",file=sys.stderr,flush=True)
    report = dict(schema="kiro.benchmark.selftest.v1",kind="deterministic-harness-controls-not-agent-benchmark",
                  passed=all(r["passed"]==r["expected"] for r in results), model_trials_run=0,
                  compiler_sha256=sha(binary), inputs=source_identity(), seconds=time.monotonic()-started,controls=results)
    if args.out:
        args.out.parent.mkdir(parents=True,exist_ok=True)
        args.out.write_text(json.dumps(report,indent=2)+"\n")
    print(json.dumps({k:v for k,v in report.items() if k not in ("controls","inputs")}))
    return 0 if report["passed"] else 1


def run(args):
    path = args.plan.resolve()
    plan = json.loads(path.read_text())
    if plan["schema"] != SCHEMA or plan["inputs"] != source_identity() or plan["compiler_sha256"] != sha(args.kiro):
        raise ValueError("plan inputs/compiler changed; prepare a new cohort")
    if not args.driver:
        raise ValueError("an external JSONL model driver is required")
    preflight = doctor(args.image,args.kiro,args.driver)
    (path.parent / "preflight.json").write_text(json.dumps(preflight,indent=2)+"\n")
    if not preflight["ready"]:
        print(json.dumps(preflight));return 2
    image = preflight["image_id"]
    settings = json.loads(args.settings.read_text()) if args.settings else {}
    driver_files={str(Path(value).resolve()):sha(value) for value in args.driver if Path(value).is_file()}
    executable=shutil.which(args.driver[0])
    if executable:driver_files[str(Path(executable).resolve())]=sha(executable)
    configuration = dict(model=args.model,settings=settings,driver=args.driver,driver_files=driver_files,image=image)
    config_path = path.parent / "configuration.json"
    if config_path.exists() and json.loads(config_path.read_text()) != configuration:
        raise ValueError("configuration changed; prepare a new cohort")
    config_path.write_text(json.dumps(configuration,indent=2)+"\n")
    records = []
    for trial in plan["schedule"]:
        folder = path.parent / "trials" / trial["id"]
        if folder.exists():
            raise ValueError("trial directory exists; use a fresh cohort (no silent resume/rerun)")
        folder.mkdir(parents=True)
        project = folder / "project"
        task = make_workspace(project,trial["task"],trial["condition"])
        before = files_hash(project)
        (folder / "before.json").write_text(json.dumps(before,indent=2)+"\n")
        report = run_session(project,folder,task,trial["condition"],image,args.driver,args.model,settings,plan["limits"])
        report.update(trial_id=trial["id"], repeat=trial["repeat"], compiler_sha256=plan["compiler_sha256"],guide=plan["guide"],plan_sha256=sha(path),configuration=configuration)
        try:
            report["evaluation"] = evaluate_isolated(project,image,trial["task"])
        except Exception as error:
            report["evaluation"] = dict(passed=False,status="evaluation_error",error=str(error))
        after = files_hash(project)
        protected = [name for name in before if name.startswith(".kiro/") or name in ("AGENTS.md","task.json")]
        report["integrity_passed"] = all(before[name]==after.get(name) for name in protected)
        report["changed_files"] = sorted(k for k in before.keys()|after.keys() if before.get(k)!=after.get(k))
        report["after_hashes"] = after
        (folder / "result.json").write_text(json.dumps(report,indent=2)+"\n")
        records.append(report)
        # Always publish interim results, including driver failures, before moving on.
        summarize(records,path.parent / "summary.json")
        print(json.dumps(dict(trial=trial["id"],kind=report["kind"],status=report["status"],passed=report["evaluation"].get("passed",False))),flush=True)
    return 0


def summarize(records,path):
    groups={}
    for condition in ("core","analysis"):
        selected=[r for r in records if r["condition"]==condition and r["kind"]=="model"]
        groups[condition]=dict(attempted_trials=sum(r["condition"]==condition for r in records),trials=len(selected),passed=sum(r["status"]=="finished" and r.get("integrity_passed",False) and r["evaluation"].get("passed",False) for r in selected),
                               agent_seconds=[r["agent_seconds"] for r in selected],
                               tool_calls=[r["tool_calls"] for r in selected],
                               token_usage=[r["usage"] for r in selected])
    report=dict(schema="kiro.benchmark.summary.v1",kind="model-trials" if any(g["trials"] for g in groups.values()) else "no-model-results",
                conditions=groups, excluded_nonmodel_records=sum(r["kind"]!="model" for r in records),
                limitations=["small task-specific samples are not a universal language ranking", "usage is adapter-reported; unavailable fields remain null", "Go comparison is not part of this cohort"])
    Path(path).write_text(json.dumps(report,indent=2)+"\n")


def image_context(args):
    if args.out.exists():
        raise ValueError("output already exists")
    args.out.mkdir(parents=True)
    shutil.copy2(args.kiro,args.out/"kiro")
    shutil.copytree(args.go_root,args.out/"go")
    (args.out/"Dockerfile").write_text('''FROM python:3.12-slim-bookworm
COPY kiro /opt/kiro
COPY go /opt/go
ENV KIRO_GO_BIN=/opt/go/bin/go CGO_ENABLED=0 GOTOOLCHAIN=local
WORKDIR /work
''')
    print(json.dumps(dict(context=str(args.out),compiler_sha256=sha(args.kiro),instruction="Build this context with Docker, then doctor the image; model/oracle sources are not included.")))


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    sub=parser.add_subparsers(dest="command",required=True)
    p=sub.add_parser("prepare");p.add_argument("--kiro",type=Path,required=True);p.add_argument("--out",type=Path,required=True)
    p.add_argument("--repetitions",type=int,default=3);p.add_argument("--seed",type=int,default=20261020)
    p=sub.add_parser("selftest");p.add_argument("--kiro",type=Path,required=True);p.add_argument("--out",type=Path)
    p=sub.add_parser("doctor");p.add_argument("--kiro",type=Path,required=True);p.add_argument("--image",required=True)
    p.add_argument("--driver",nargs=argparse.REMAINDER)
    p=sub.add_parser("image-context");p.add_argument("--kiro",type=Path,required=True);p.add_argument("--go-root",type=Path,required=True);p.add_argument("--out",type=Path,required=True)
    p=sub.add_parser("run");p.add_argument("--plan",type=Path,required=True);p.add_argument("--kiro",type=Path,required=True);p.add_argument("--image",required=True)
    p.add_argument("--model",required=True);p.add_argument("--settings",type=Path);p.add_argument("--driver",nargs=argparse.REMAINDER,required=True)
    args=parser.parse_args()
    if args.command=="prepare" and not 1<=args.repetitions<=20:parser.error("repetitions must be 1..20")
    try:
        if args.command=="prepare":prepare(args)
        elif args.command=="selftest":return selftest(args)
        elif args.command=="doctor":
            result=doctor(args.image,args.kiro,args.driver);print(json.dumps(result));return 0 if result["ready"] else 2
        elif args.command=="image-context":image_context(args)
        elif args.command=="run":return run(args)
    except (ValueError,OSError,subprocess.SubprocessError,json.JSONDecodeError) as error:
        print(json.dumps(dict(schema="kiro.benchmark.error.v1",error=str(error))),file=sys.stderr);return 2
    return 0


if __name__=="__main__":
    sys.exit(main())
