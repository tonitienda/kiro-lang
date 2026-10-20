"""Fresh JSONL driver sessions, exact transcript retention and bounded trial budgets."""
import hashlib
import json
import os
from pathlib import Path
import selectors
import signal
import subprocess
import time
import tempfile

from broker import Broker, DockerSandbox
from evaluate import command, compare_response, finalize
from oracle import cases, structure, api_contracts

TOOLS = {
    "list": {"path": "workspace-relative directory"},
    "read": {"path": "workspace-relative file", "offset": "optional byte offset", "length": "optional byte count <=262144"},
    "write": {"path": "editable .ki/.md/.json file", "content": "complete replacement text <=262144 bytes"},
    "delete": {"path": "editable file"},
    "kiro": {"args": "argv array, no shell; fixed /opt/kiro", "save_as": "optional complete JSON output path"},
}


class Driver:
    def __init__(self, argv, stderr, timeout):
        self.process = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        stderr=stderr, start_new_session=True)
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.process.stdout, selectors.EVENT_READ)
        os.set_blocking(self.process.stdin.fileno(), False)
        self.buffer = b""
        self.timeout = timeout

    def exchange(self, request, remaining):
        deadline = time.monotonic() + min(self.timeout, remaining)
        data = memoryview((json.dumps(request) + "\n").encode())
        with selectors.DefaultSelector() as writer:
            writer.register(self.process.stdin, selectors.EVENT_WRITE)
            while data:
                wait = deadline - time.monotonic()
                if wait <= 0 or not writer.select(wait):
                    raise TimeoutError("model driver input timeout")
                try:
                    sent = os.write(self.process.stdin.fileno(), data)
                    data = data[sent:]
                except BlockingIOError:
                    continue
        while b"\n" not in self.buffer:
            wait = deadline - time.monotonic()
            if wait <= 0 or not self.selector.select(wait):
                raise TimeoutError("model driver response timeout")
            block = os.read(self.process.stdout.fileno(), 65536)
            if not block:
                raise RuntimeError("model driver closed stdout")
            self.buffer += block
            if len(self.buffer) > 1048576:
                raise ValueError("driver response exceeds 1 MiB")
        line, self.buffer = self.buffer.split(b"\n", 1)
        value = json.loads(line)
        if not isinstance(value, dict):
            raise ValueError("driver response must be a JSON object")
        return value

    def close(self):
        try:
            os.killpg(self.process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        self.process.wait()
        self.selector.close()
        self.process.stdin.close()
        self.process.stdout.close()


def run_session(project, trial_dir, task, condition, image, driver_argv, model, settings, limits):
    start = time.monotonic()
    deadline = start + limits["trial_seconds"]
    out = Path(trial_dir)
    report = dict(schema="kiro.benchmark.trial.v1", task=task["id"], condition=condition,
                  status="driver_error", kind="unidentified", model=model, settings=settings,
                  tool_calls=0, usage=None, evaluation=None)
    transcript = out / "transcript.jsonl"
    observations, usage = [], []
    def record(direction, payload):
        with transcript.open("a") as file:
            file.write(json.dumps(dict(direction=direction, elapsed_seconds=time.monotonic()-start, payload=payload)) + "\n")
    def exchange(driver, message):
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError("trial time budget exhausted")
        record("to_driver", message)
        reply = driver.exchange(message, remaining)
        record("from_driver", reply)
        return reply
    with (out / "driver.stderr").open("wb") as stderr:
        driver = None
        broker = None
        try:
            with DockerSandbox(image, project) as sandbox:
                broker = Broker(project, condition, sandbox, max_calls=limits["tool_calls"], tool_timeout=limits["tool_seconds"])
                driver = Driver(driver_argv, stderr, limits["response_seconds"])
                bootstrap = (Path(project) / "AGENTS.md").read_text()
                reply = exchange(driver, dict(type="start", schema="kiro.benchmark.driver.v1", model=model, settings=settings,
                     condition=condition, task=task, bootstrap=bootstrap, tools=TOOLS,
                     allowed_kiro=["describe", "symbols", "fmt", "check", "lint", "test"] + (["analyze"] if condition=="analysis" else []),
                     limits=limits, workspace="/work", instructions="Use only the provided tools. Read the Kiro guide before editing. Return identity first, then one action per observation, or finish. Do not request human source review."))
                if reply.get("type") != "identity" or reply.get("model") != model or reply.get("execution_kind") not in ("model", "scripted-control"):
                    raise ValueError("driver must identify the requested model and execution_kind")
                report["kind"] = reply["execution_kind"]
                report["backend_identity"] = reply
                next_message = dict(type="ready")
                while True:
                    reply = exchange(driver, next_message)
                    usage.append(reply.get("usage"))
                    if reply.get("type") == "finish":
                        report["status"] = "finished"
                        report["final_message"] = reply.get("message", "")
                        break
                    if reply.get("type") != "action":
                        raise ValueError("expected action or finish")
                    action = reply.get("action")
                    attempt = dict(tool=action.get("tool") if isinstance(action,dict) else None, ok=False, seconds=None)
                    if isinstance(action,dict) and action.get("tool")=="kiro" and isinstance(action.get("args"),list):
                        attempt["command"] = action["args"][0] if action["args"] else None
                        attempt["operation"] = action["args"][1] if len(action["args"])>1 and action["args"][0]=="analyze" else None
                    observations.append(attempt)
                    observation = broker.execute(action)
                    attempt.update(ok=observation["ok"], seconds=observation["seconds"])
                    next_message = dict(type="observation", observation=observation)
                report["tool_calls"] = broker.calls
                report["file_read_bytes"] = broker.read_bytes
                report["file_reads"] = broker.file_reads
                report["tool_output_bytes"] = broker.output_bytes
        except TimeoutError as error:
            report["status"] = "budget_exhausted"
            report["error"] = str(error)
        except Exception as error:
            report["error"] = str(error)
        finally:
            if driver:
                driver.close()
            if broker:
                report["tool_calls"] = broker.calls
                report["file_read_bytes"] = broker.read_bytes
                report["file_reads"] = broker.file_reads
                report["tool_output_bytes"] = broker.output_bytes
    report["agent_seconds"] = time.monotonic() - start
    report["tool_observations"] = observations
    pending = set()
    repairs = 0
    for observed in observations:
        cmd = observed.get("command")
        if cmd in ("check","lint","test"):
            if not observed["ok"]:
                pending.add(cmd)
            elif cmd in pending:
                repairs += 1
                pending.remove(cmd)
    report["verification_recovery_cycles"] = repairs
    report["documentation_read_bytes"] = sum(read["bytes"] for read in report.get("file_reads",[]) if read["path"].startswith(".kiro/"))
    # Usage is trusted adapter telemetry, not inferred from source size. Keep raw entries.
    report["usage_entries"] = usage
    if usage and all(isinstance(item, dict) and all(type(item.get(k)) is int and item[k]>=0 for k in ("input_tokens","output_tokens")) for item in usage):
        report["usage"] = {key:sum(item[key] for item in usage) for key in ("input_tokens","output_tokens")}
        report["usage_source"] = "driver_reported"
    else:
        report["usage_source"] = "unavailable"
    report["cost"] = None
    report["transcript_sha256"] = hashlib.sha256(transcript.read_bytes()).hexdigest() if transcript.exists() else None
    return report


# This transport sees requests and actual responses only, never expected values/oracle code.
HTTP_TRANSPORT = r"""
import sys,json,urllib.request,urllib.error
for line in sys.stdin:
 try:
  value=json.loads(line)
  req=urllib.request.Request("http://127.0.0.1:8080"+value["path"],data=value["body"].encode(),method=value["method"],headers={"Content-Type":"application/json"})
  try: response=urllib.request.urlopen(req,timeout=1)
  except urllib.error.HTTPError as error: response=error
  with response:
   body=response.read(65537)
   if len(body)>65536: raise ValueError("response too large")
   result={"status":response.status,"body":body.decode("utf-8")}
 except Exception as error: result={"error":str(error)}
 print(json.dumps(result),flush=True)
"""


def evaluate_isolated(project, image, task):
    start=time.monotonic()
    deadline=start+150
    report=dict(schema="kiro.benchmark.evaluation.v1",task=task,passed=False,status="verification_failed",
                steps=[],cases=[],structural=dict(required=False,passed=True,checks=[]),
                kind="deterministic-evaluation-not-model-performance")
    # No oracle files or expected values enter the candidate container, even at evaluation.
    with DockerSandbox(image, project) as sandbox:
        for args in (["lint",".","--json"],["test","."],["build",".","--out","/tmp/service"]):
            step=sandbox.kiro(args,min(60,max(1,deadline-time.monotonic())))
            report["steps"].append(step)
            if step["exit_code"]:
                return finalize(report,task,start)
        step=sandbox.kiro(["symbols",".","--json"],30)
        report["steps"].append(step)
        try:
            report["api"]=api_contracts(json.loads(step["stdout"])["data"]["symbols"])
        except (ValueError,KeyError):
            report["api"]=dict(passed=False,error="public API unavailable")
        result=command(["docker","exec","-d","--env","ADDR=127.0.0.1:8080",sandbox.name,"/tmp/service"],str(project),10)
        if result["exit_code"]:
            report["status"]="service_start_failed"
            return finalize(report,task,start)
        with tempfile.TemporaryFile() as stderr:
            transport=Driver(["docker","exec","-i",sandbox.name,"python3","-u","-c",HTTP_TRANSPORT],stderr,3)
            try:
                for attempt in range(30):
                    reply=transport.exchange(dict(path="/health",method="GET",body=""),max(0.01,deadline-time.monotonic()))
                    if reply.get("status")==200:break
                    time.sleep(0.02)
                else:
                    report["status"]="service_start_failed"
                    return finalize(report,task,start)
                for case in cases(task):
                    if time.monotonic()>=deadline:
                        report["status"]="timeout"
                        break
                    reply=transport.exchange({key:case[key] for key in ("path","method","body")},deadline-time.monotonic())
                    try:
                        if "error" in reply:raise ValueError(reply["error"])
                        report["cases"].append(compare_response(case,reply["status"],reply["body"]))
                    except (ValueError,KeyError) as error:
                        report["cases"].append(dict(id=case["id"],group=case["group"],passed=False,error=str(error)))
                else:
                    report["status"]="complete"
            finally:
                transport.close()
        if task in ("payment-boundary","payment-declined"):
            step=sandbox.kiro(["analyze","graph",".","--json"],30)
            report["steps"].append(step)
            if step["exit_code"] or step["truncated"]:
                report["structural"]=dict(required=True,passed=False,error="graph unavailable")
            else:
                report["structural"]=structure(json.loads(step["stdout"])["data"]["result"],task)
    return finalize(report,task,start)
