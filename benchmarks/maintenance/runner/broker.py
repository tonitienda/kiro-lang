"""A bounded tool surface for a fresh model driver; no arbitrary host shell.

The model driver is trusted transport code. Model actions are mediated here. Candidate
Kiro runs only inside a network-disabled Docker container; evaluator files are absent.
"""
import json
import os
from pathlib import Path
import shutil
import subprocess
import time
import tempfile
import uuid

from evaluate import command

MAX_FILE = 262144
MAX_OUTPUT = 131072


class DockerSandbox:
    def __init__(self, image, project):
        self.name = "kiro-bench-" + uuid.uuid4().hex
        self.project = Path(project).resolve()
        self.image = image

    def __enter__(self):
        argv = ["docker", "create", "--name", self.name, "--network", "none", "--read-only",
                "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "256",
                "--memory", "1536m", "--cpus", "1", "--user", f"{os.getuid()}:{os.getgid()}",
                "--mount", f"type=bind,src={self.project},dst=/work",
                "--tmpfs", "/tmp:rw,exec,nosuid,size=768m,mode=1777",
                "--tmpfs", "/cache:rw,nosuid,size=384m,mode=1777",
                "--env", "HOME=/tmp", "--env", "GOCACHE=/cache/go", "--env", "CGO_ENABLED=0",
                "--env", "GOTOOLCHAIN=local", "--env", "KIRO_GO_BIN=/opt/go/bin/go",
                "--workdir", "/work", self.image, "sleep", "infinity"]
        for name in (".kiro", "AGENTS.md", "task.json"):
            pinned = self.project / name
            if pinned.exists():
                argv[2:2] = ["--mount", f"type=bind,src={pinned},dst=/work/{name},readonly"]
        result = command(argv, str(self.project), 30)
        if result["exit_code"] or result["timed_out"]:
            raise RuntimeError("isolated container creation failed: " + result["stderr"])
        result = command(["docker", "start", self.name], str(self.project), 30)
        if result["exit_code"]:
            self.close()
            raise RuntimeError("isolated container start failed: " + result["stderr"])
        return self

    def close(self):
        command(["docker", "rm", "-f", self.name], str(self.project), 15)

    def __exit__(self, *_):
        self.close()

    def kiro(self, arguments, timeout):
        result = command(["docker", "exec", self.name, "/opt/kiro", *arguments], str(self.project), timeout, max_output=8388608 if arguments[:2] == ["analyze", "graph"] else MAX_OUTPUT)
        if result["timed_out"]:
            # docker exec client termination alone does not terminate the child process.
            self.close()
            raise TimeoutError("tool timeout; trial container terminated")
        return result


def doctor(image, binary, driver=None):
    checks = []
    def add(name, passed, detail):
        checks.append(dict(name=name, passed=bool(passed), detail=detail))
    docker = shutil.which("docker")
    add("docker-cli", docker, docker or "not installed")
    info = None
    if docker:
        result = command([docker, "image", "inspect", image, "--format", "{{.Id}}"], str(Path.cwd()), 15)
        image_id = result["stdout"].strip()
        add("pinned-image", result["exit_code"] == 0 and image_id.startswith("sha256:"), image_id or result["stderr"])
        if result["exit_code"] == 0 and image_id.startswith("sha256:"):
            result = command([docker, "run", "--rm", "--network", "none", "--read-only", "--cap-drop", "ALL",
                              "--security-opt", "no-new-privileges", image_id, "sha256sum", "/opt/kiro"], str(Path.cwd()), 30)
            import hashlib
            expected = hashlib.sha256(Path(binary).read_bytes()).hexdigest()
            add("image-compiler", result["exit_code"] == 0 and result["stdout"].split()[:1] == [expected], result["stdout"].strip() or result["stderr"])
            info = image_id
            try:
                with tempfile.TemporaryDirectory(prefix="kiro-isolation-probe-") as tmp:
                    root=Path(tmp)/"project";root.mkdir()
                    (root/"AGENTS.md").write_text("pinned")
                    canary=Path(tmp)/"withheld-canary";canary.write_text("not mounted")
                    with DockerSandbox(image_id,root) as sandbox:
                        probe = r"""import json,os,socket,sys
checks={"workspace":open("/work/AGENTS.md").read()=="pinned","host_canary_absent":not os.path.exists(sys.argv[1]),"provider_key_absent":not bool(os.getenv("OPENAI_API_KEY"))}
try:
 open("/work/AGENTS.md","w").write("changed")
 checks["pinned_readonly"]=False
except OSError: checks["pinned_readonly"]=True
try:
 with socket.create_connection(("192.0.2.1",80),timeout=0.2):pass
 checks["outbound_blocked"]=False
except OSError: checks["outbound_blocked"]=True
print(json.dumps(checks))
"""
                        checked=command([docker,"exec",sandbox.name,"python3","-c",probe,str(canary)],str(root),15)
                        flags=json.loads(checked["stdout"]) if checked["exit_code"]==0 else {}
                        add("runtime-isolation-probe",bool(flags) and all(flags.values()),flags or checked["stderr"])
            except (OSError,ValueError,RuntimeError) as error:
                add("runtime-isolation-probe",False,str(error))
    executable = shutil.which(driver[0]) if driver else None
    add("model-driver", executable, executable or "no usable external model driver configured")
    return dict(schema="kiro.benchmark.preflight.v1", ready=all(c["passed"] for c in checks),
                checks=checks, image_id=info, model_trials_run=0)


class Broker:
    def __init__(self, root, condition, sandbox, max_calls=120, tool_timeout=60):
        self.root = Path(root).resolve()
        if condition not in ("core", "analysis"):
            raise ValueError("unknown condition")
        self.condition, self.sandbox = condition, sandbox
        self.max_calls, self.tool_timeout = max_calls, tool_timeout
        self.calls = 0
        self.read_bytes = 0
        self.file_reads = []
        self.output_bytes = 0

    def path(self, raw):
        if not isinstance(raw, str) or not raw or len(raw) > 4096:
            raise ValueError("invalid workspace path")
        p = (self.root / raw).resolve()
        if not p.is_relative_to(self.root):
            raise ValueError("path is outside workspace")
        return p

    def editable(self, raw):
        p = self.path(raw)
        rel = p.relative_to(self.root)
        if not rel.parts or rel.parts[0] == ".kiro" or str(rel) in ("AGENTS.md", "task.json"):
            raise ValueError("pinned instructions/docs are read-only")
        if p.suffix not in (".ki", ".json", ".md"):
            raise ValueError("only source, JSON and Markdown files may be edited")
        return p

    def execute(self, action):
        self.calls += 1
        if self.calls > self.max_calls:
            raise TimeoutError("tool-call budget exhausted")
        start = time.monotonic()
        try:
            if not isinstance(action, dict):
                raise ValueError("action must be an object")
            tool = action.get("tool")
            if tool == "read":
                p = self.path(action["path"])
                raw = p.read_bytes()
                if len(raw) > MAX_FILE and "length" not in action:
                    raise ValueError("file exceeds read limit; use smaller files")
                offset, length = action.get("offset", 0), action.get("length", len(raw))
                if type(offset) is not int or type(length) is not int or offset < 0 or length < 0 or length > MAX_FILE:
                    raise ValueError("invalid read range")
                chunk = raw[offset:offset + length]
                self.read_bytes += len(chunk)
                self.file_reads.append(dict(path=str(p.relative_to(self.root)), bytes=len(chunk), offset=offset))
                result = dict(ok=True, content=chunk.decode("utf-8", errors="replace"), bytes=len(chunk), total_bytes=len(raw), offset=offset)
            elif tool == "list":
                p = self.path(action.get("path", "."))
                names = []
                for child in sorted(p.iterdir()):
                    if len(names) >= 500:
                        raise ValueError("directory exceeds listing limit")
                    self.path(str(child))
                    names.append(dict(name=child.name, directory=child.is_dir()))
                result = dict(ok=True, entries=names)
            elif tool == "write":
                p = self.editable(action["path"])
                content = action["content"]
                if not isinstance(content, str) or len(content.encode()) > MAX_FILE:
                    raise ValueError("file exceeds write limit")
                if p.exists() and p.is_dir():
                    raise ValueError("target is a directory")
                # A per-project bound keeps generated source and graph inputs finite.
                size = sum(f.stat().st_size for f in self.root.rglob("*") if f.is_file())
                if size + len(content.encode()) > 8 * 1024 * 1024:
                    raise ValueError("workspace size budget exhausted")
                p.parent.mkdir(parents=True, exist_ok=True)
                p.write_text(content)
                result = dict(ok=True, bytes=len(content.encode()))
            elif tool == "delete":
                p = self.editable(action["path"])
                p.unlink()
                result = dict(ok=True)
            elif tool == "kiro":
                args = action["args"]
                if not isinstance(args, list) or not args or len(args) > 40 or any(not isinstance(a, str) or len(a) > 4096 for a in args):
                    raise ValueError("invalid command arguments")
                allowed = {"describe", "symbols", "fmt", "check", "lint", "test"}
                if self.condition == "analysis":
                    allowed.add("analyze")
                if args[0] not in allowed:
                    raise ValueError("command unavailable in this condition")
                result = self.sandbox.kiro(args, self.tool_timeout)
                result["ok"] = result["exit_code"] == 0
                if "save_as" in action:
                    dest = self.editable(action["save_as"])
                    if dest.suffix != ".json" or result.get("truncated"):
                        raise ValueError("save_as requires complete JSON output and a .json path")
                    json.loads(result["stdout"])
                    dest.parent.mkdir(parents=True, exist_ok=True)
                    if len(result["stdout"].encode()) + sum(f.stat().st_size for f in self.root.rglob("*") if f.is_file()) > 8 * 1024 * 1024:
                        raise ValueError("workspace size budget exhausted")
                    dest.write_text(result["stdout"])
                if len(result.get("stdout", "").encode()) > MAX_OUTPUT:
                    result["stdout_bytes"] = len(result["stdout"].encode())
                    result["stdout"] = result["stdout"].encode()[:MAX_OUTPUT].decode(errors="replace")
                    result["truncated"] = True
            else:
                raise ValueError("unknown tool")
        except TimeoutError:
            raise
        except (ValueError, KeyError, OSError, UnicodeError) as error:
            result = dict(ok=False, error=str(error))
        result["seconds"] = time.monotonic() - start
        self.output_bytes += len(json.dumps(result).encode())
        return result
