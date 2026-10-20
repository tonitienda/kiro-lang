"""Run with python3 -B -m unittest discover -s benchmarks/maintenance/runner -v."""
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import benchmark
from broker import Broker, DockerSandbox
from oracle import cases, checkout_expected, structure, api_contracts
from evaluate import compare_response
from session import run_session


class FakeSandbox:
    def __init__(self,*args,**kwargs):
        self.arguments=[]
    def __enter__(self):return self
    def __exit__(self,*args):pass
    def kiro(self,args,timeout):
        self.arguments.append(args)
        return dict(exit_code=0,stdout='{"ok":true}',stderr="",timed_out=False,truncated=False,seconds=0)


class BrokerTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.root=Path(self.tmp.name)/"project";self.root.mkdir()
        (self.root/"AGENTS.md").write_text("instructions")
        (self.root/".kiro/skill").mkdir(parents=True)
        (self.root/".kiro/skill/SKILL.md").write_text("guide")
        self.sandbox=FakeSandbox()
        self.broker=Broker(self.root,"core",self.sandbox)
    def tearDown(self):self.tmp.cleanup()
    def test_paths_and_immutable_instructions(self):
        outside=Path(self.tmp.name)/"oracle.py";outside.write_text("not visible")
        (self.root/"escape").symlink_to(outside)
        for name in ("../oracle.py",str(outside),"escape"):
            self.assertFalse(self.broker.execute(dict(tool="read",path=name))["ok"])
        for name in ("AGENTS.md","task.json",".kiro/skill/SKILL.md"):
            self.assertFalse(self.broker.execute(dict(tool="write",path=name,content="overwrite"))["ok"])
        self.assertEqual((self.root/"AGENTS.md").read_text(),"instructions")
        self.assertTrue(self.broker.execute(dict(tool="read",path=".kiro/skill/SKILL.md"))["ok"])
    def test_tool_ablation_is_enforced(self):
        for args in (["analyze","graph","."],["__analysis-test","input","out"],["run","."],["build","."]):
            self.assertFalse(self.broker.execute(dict(tool="kiro",args=args))["ok"])
        self.assertEqual(self.sandbox.arguments,[])
        analysis=Broker(self.root,"analysis",self.sandbox)
        self.assertTrue(analysis.execute(dict(tool="kiro",args=["analyze","metrics",".","--json"]))["ok"])
        self.assertEqual(len(self.sandbox.arguments),1)
    def test_edits_and_budget(self):
        self.assertTrue(self.broker.execute(dict(tool="write",path="internal/new/main.ki",content="mod new"))["ok"])
        self.assertEqual(self.broker.execute(dict(tool="read",path="internal/new/main.ki"))["content"],"mod new")
        self.assertTrue(self.broker.execute(dict(tool="delete",path="internal/new/main.ki"))["ok"])
        self.assertFalse(self.broker.execute(dict(tool="shell",command="ls"))["ok"])
        bounded=Broker(self.root,"core",self.sandbox,max_calls=1)
        bounded.execute(dict(tool="list"))
        with self.assertRaises(TimeoutError):bounded.execute(dict(tool="list"))
    def test_container_mounts_are_separate(self):
        commands=[]
        def record(args,*a,**kw):
            commands.append(args)
            return dict(exit_code=0,timed_out=False,stdout="",stderr="")
        with patch("broker.command",record):
            with DockerSandbox("sha256:pinned",self.root):pass
        create=[args for args in commands if args[1]=="create"]
        self.assertEqual(create[0][create[0].index("--network")+1],"none")
        self.assertIn("--read-only",create[0]);self.assertIn("no-new-privileges",create[0])
        self.assertFalse(any("dst=/oracle" in item for item in create[0]))
        self.assertTrue(any("dst=/work/.kiro,readonly" in item for item in create[0]))
        self.assertFalse(any("src=/," in item for item in create[0]))


class OracleTests(unittest.TestCase):
    def test_discount_threshold_and_fee(self):
        base=dict(sku="notebook",units=10,available=12,delivery="standard",coupon="SAVE10",payment="card")
        status,receipt=checkout_expected(base,"discount-policy")
        self.assertEqual(status,200);self.assertEqual(receipt["discount_cents"],500)
        self.assertEqual(receipt["total_cents"],4830);self.assertEqual(receipt["payment_reference"],"card:4800")
        self.assertEqual(checkout_expected(base|dict(units=9),"discount-policy")[1]["discount_cents"],0)
        self.assertEqual(checkout_expected(base,"baseline"),(422,"unknown coupon: SAVE10"))
    def test_error_priority_and_case_identities(self):
        body=dict(sku="book",units=0,available=0,delivery="standard",payment="declined",coupon=None)
        self.assertEqual(checkout_expected(body,"payment-declined"),(422,"invalid quantity: 0"))
        for task in (*benchmark.TASKS,"baseline"):
            selected=cases(task)
            self.assertEqual(len({c["id"] for c in selected}),len(selected))
            self.assertTrue(any(c["group"]=="strict-schema" for c in selected))
    def test_public_key_contracts_and_exact_json_types(self):
        symbols=json.loads((Path(__file__).parent/"contracts.json").read_text())
        self.assertTrue(api_contracts(symbols)["passed"])
        authorize=next(node for node in symbols if node["id"]=="checkout.authorize")
        authorize["parameters"][0]["name"]="payment"
        self.assertFalse(api_contracts(symbols)["passed"])
        case=dict(id="typed",group="typed",status=200,expected={"active":True})
        self.assertFalse(compare_response(case,200,'{"active":1}')["passed"])
    def test_unused_payment_module_does_not_satisfy_boundary(self):
        report=structure(dict(nodes=[dict(id="payments.unused",module="payments")],edges=[]),"payment-boundary")
        self.assertFalse(report["passed"])


class SessionTests(unittest.TestCase):
    def test_scripted_driver_is_logged_and_excluded_from_model_results(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp=Path(tmp);project=tmp/"project"
            task=benchmark.make_workspace(project,"reservation-expiry","core")
            driver=tmp/"driver.py"
            driver.write_text('''import sys,json
for line in sys.stdin:
 message=json.loads(line)
 if message['type']=='start': reply={'type':'identity','execution_kind':'scripted-control','model':message['model']}
 elif message['type']=='ready': reply={'type':'action','action':{'tool':'read','path':'AGENTS.md'}}
 else: reply={'type':'finish','message':'control complete'}
 print(json.dumps(reply),flush=True)
''')
            with patch("session.DockerSandbox",FakeSandbox):
                report=run_session(project,tmp,task,"core","unused",[sys.executable,"-B",str(driver)],"test-transport",{},dict(trial_seconds=10,response_seconds=5,tool_calls=5,tool_seconds=1))
            self.assertEqual(report["status"],"finished")
            self.assertEqual(report["kind"],"scripted-control")
            self.assertEqual(report["tool_calls"],1)
            self.assertEqual(report["usage_source"],"unavailable")
            self.assertEqual(report["transcript_sha256"],hashlib.sha256((tmp/"transcript.jsonl").read_bytes()).hexdigest())
            report["evaluation"]={"passed":False}
            path=tmp/"summary.json";benchmark.summarize([report],path)
            summary=json.loads(path.read_text())
            self.assertEqual(summary["kind"],"no-model-results")
            self.assertEqual(summary["conditions"]["core"]["trials"],0)
    def test_environment_missing_docker_is_blocked(self):
        with patch("broker.shutil.which",return_value=None):
            report=benchmark.doctor("missing","missing")
        self.assertFalse(report["ready"])
        self.assertEqual(report["model_trials_run"],0)


if __name__=="__main__":unittest.main()
