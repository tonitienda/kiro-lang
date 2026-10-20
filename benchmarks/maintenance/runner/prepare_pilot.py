#!/usr/bin/env python3
"""Versioned native pilot preparation. Does not run or evaluate model candidates."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import time

SUITE = Path(__file__).resolve().parents[1]
ROOT = SUITE.parents[1]


def hashes(root):
    return {p.relative_to(root).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in sorted(root.rglob('*')) if p.is_file()}


def prompt(language, task):
    bootstrap = (
        'This project uses Kiro. Read AGENTS.md, task.json, .kiro/skill/SKILL.md and '
        '.kiro/skill/references/kiro.json before editing. Use '
        '.kiro/skill/references/language.md as the primary language reference, reading '
        'the sections needed for this task. Use kiro describe --json and kiro symbols . '
        '--json for capabilities and contracts. Verify with `kiro fmt .`, `kiro check .`, '
        '`kiro lint .` and `kiro test .`. Analysis is unavailable in this core pilot.'
        if language == 'kiro' else
        'This project uses Go. Read AGENTS.md and task.json before editing. Inspect the '
        'relevant packages and public tests. Use the standard library and existing module; '
        'no Kiro reference or tools are required. Verify with `gofmt`, `go test ./...` and '
        '`go vet ./...`. Do not fetch external dependencies.'
    )
    return f'''Work only in the supplied fresh {language} service workspace.
{bootstrap}

Implement task {task['id']} version {task['version']}:
{task['requirement']}

Make only this independent maintenance change. Preserve unrelated behavior, public APIs
and existing tests; add meaningful tests for the change. Do not alter AGENTS.md, task.json,
vendored guidance or toolchain. Implement, review and verify using the declared tools;
do not wait for a human source-review gate. Stop when the implementation is complete.

Use only the provisioned isolation environment. Do not access the parent repository,
evaluator, scripted fixes, other trials or network. If isolation is unavailable, report an
environment blocker; do not execute candidate code on the host as a fallback.

Report the change, verification commands and actual observed outcomes. Record available
implementation wall time, documentation/context acquisition, tool use and telemetry.
Leave unavailable token/cost/time fields unavailable. Preparation and independent evaluation
are separate measurements. An independent evaluator checks behavioral, public API and
structural outcomes after the session; project tests alone do not establish success.
'''


def prepare(out, language, task_id, kiro=None):
    started = time.monotonic()
    out = Path(out).resolve()
    if out.exists():
        raise ValueError('output already exists; use a fresh destination')
    task = json.loads((SUITE / ('tasks' if language == 'kiro' else 'tasks-go-v1') / (task_id + '.json')).read_text())
    if language == 'kiro' and kiro is None:
        raise ValueError('--kiro is required for a scaffolded Kiro pilot')
    out.mkdir(parents=True)
    project = out / 'service'
    try:
        if language == 'kiro':
            binary = str(Path(kiro).resolve())
            subprocess.run([binary, 'new', 'service'], cwd=out, check=True, capture_output=True)
            guidance = {k:v for k,v in hashes(project).items() if k == 'AGENTS.md' or k.startswith('.kiro/')}
            for path in list(project.iterdir()):
                if path.name in ('AGENTS.md', '.kiro'): continue
                if path.is_dir(): shutil.rmtree(path)
                else: path.unlink()
            shutil.copytree(SUITE / 'workload', project, dirs_exist_ok=True)
            if guidance != {k:v for k,v in hashes(project).items() if k == 'AGENTS.md' or k.startswith('.kiro/')} :
                raise ValueError('generated scaffold guidance changed during overlay')
            toolchain = {'kiro_sha256':hashlib.sha256(Path(binary).read_bytes()).hexdigest(),
                         'describe':json.loads(subprocess.check_output([binary,'describe','--json'],text=True)),
                         'scaffold_metadata':json.loads((project/'.kiro/version.json').read_text())}
        else:
            shutil.copytree(SUITE / 'workload-go-v1', project)
            (project/'AGENTS.md').write_text('''# Go maintenance pilot
Read task.json and relevant Go packages/tests before editing. This is a stateless HTTP
checkout service using only the standard library. Main starts the server; app owns HTTP
and strict request validation; internal packages own business responsibilities.
Use ordinary Go errors, packages and types. Preserve exported contracts, JSON tags,
validation priority and unrelated behavior. Aliases/wrappers may preserve compatibility.
Run gofmt on changed Go files, go test ./... and go vet ./.... No Kiro docs are required.
Do not alter AGENTS.md, task.json or module identity. Add meaningful task tests.
Use only the declared isolated environment; no network or evaluator/other-trial access.
Implement, audit and verify without a human review gate.
''')
            toolchain = {'go_version':subprocess.check_output(['go','version'],text=True).strip()}
        (project/'task.json').write_text(json.dumps(task,indent=2)+'\n')
        (out/'PROMPT.md').write_text(prompt(language,task))
        record = dict(schema='kiro.benchmark.native-pilot-preparation.v2',kind='preparation-not-model-result',
                      language=language,task=task['id'],task_version=task['version'],
                      fixture='kiro-scaffold-pilot-v2' if language=='kiro' else 'go-maintenance-v1',
                      preparation_seconds=time.monotonic()-started,toolchain=toolchain,
                      workspace_sha256=hashes(project),prompt_sha256=hashlib.sha256((out/'PROMPT.md').read_bytes()).hexdigest(),
                      model_trials_run=0,isolation_verified=False,
                      source_commit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
                      source_dirty=bool(subprocess.check_output(['git','status','--porcelain'],cwd=ROOT)))
        (out/'preparation.json').write_text(json.dumps(record,indent=2)+'\n')
    except Exception:
        (out/'PREPARATION_FAILED').write_text('Preparation failed; use a fresh destination after diagnosis.\n')
        raise
    return record


if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--language',choices=('go','kiro'),required=True)
    parser.add_argument('--task',choices=('discount-policy','reservation-expiry','partial-refund','payment-boundary','payment-declined'),default='payment-boundary')
    parser.add_argument('--kiro',type=Path)
    parser.add_argument('--out',type=Path,required=True)
    args=parser.parse_args()
    result=prepare(args.out,args.language,args.task,args.kiro)
    print(json.dumps({k:v for k,v in result.items() if k not in ('workspace_sha256','toolchain')}))
