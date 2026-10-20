"""Preparation checks use mocked toolchain processes; no candidate code is executed."""
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import prepare_pilot


class PreparationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.suite = self.root / 'suite'
        for directory in ('tasks', 'tasks-go-v1', 'workload', 'workload-go-v1'):
            (self.suite / directory).mkdir(parents=True)
        task = dict(id='payment-boundary', version=1, requirement='Extract payment policy.')
        for directory in ('tasks', 'tasks-go-v1'):
            (self.suite / directory / 'payment-boundary.json').write_text(json.dumps(task))
        (self.suite / 'workload/main.ki').write_text('mod main\n')
        (self.suite / 'workload-go-v1/main.go').write_text('package main\n')
        self.binary = self.root / 'kiro'
        self.binary.write_bytes(b'trusted compiler fixture')
        self.out = self.root / 'pilot'
        self.patches = [patch.object(prepare_pilot, 'SUITE', self.suite),
                        patch.object(prepare_pilot, 'ROOT', self.root),
                        patch.object(prepare_pilot.subprocess, 'run', self.scaffold),
                        patch.object(prepare_pilot.subprocess, 'check_output', self.output)]
        for item in self.patches: item.start()
    def tearDown(self):
        for item in reversed(self.patches): item.stop()
        self.tmp.cleanup()
    def scaffold(self, argv, cwd, **kwargs):
        self.assertEqual(argv[1:], ['new', 'service'])
        project = Path(cwd) / 'service'
        (project / '.kiro/skill/references').mkdir(parents=True)
        (project / 'AGENTS.md').write_text('original scaffold instructions')
        (project / '.kiro/skill/SKILL.md').write_text('original skill')
        (project / '.kiro/skill/references/language.md').write_text('authoritative reference')
        (project / '.kiro/version.json').write_text('{"guide_version":"test-guide"}')
        (project / 'main.ki').write_text('scaffold startup must be replaced')
        (project / 'obsolete').mkdir()
        (project / 'obsolete/old.ki').write_text('remove old source')
    def output(self, argv, **kwargs):
        if argv[0] == 'git':
            return 'abc123\n' if argv[1] == 'rev-parse' else b'?? untracked\n'
        if argv[0] == 'go': return 'go version go1.test\n'
        return '{"ok":true}'
    def test_kiro_preserves_fresh_scaffold_guidance_and_records_hashes(self):
        record = prepare_pilot.prepare(self.out, 'kiro', 'payment-boundary', self.binary)
        project = self.out / 'service'
        self.assertEqual((project / 'AGENTS.md').read_text(), 'original scaffold instructions')
        self.assertEqual((project / '.kiro/skill/SKILL.md').read_text(), 'original skill')
        self.assertEqual((project / 'main.ki').read_text(), 'mod main\n')
        self.assertFalse((project / 'obsolete').exists())
        self.assertEqual(record['workspace_sha256'], prepare_pilot.hashes(project))
        self.assertEqual(record['prompt_sha256'], hashlib.sha256((self.out / 'PROMPT.md').read_bytes()).hexdigest())
        self.assertEqual(record['toolchain']['kiro_sha256'], hashlib.sha256(self.binary.read_bytes()).hexdigest())
        self.assertEqual(record['toolchain']['scaffold_metadata']['guide_version'], 'test-guide')
        self.assertEqual(record['model_trials_run'], 0)
        self.assertFalse(record['isolation_verified'])
        self.assertTrue(record['source_dirty'])
        self.assertEqual(json.loads((project / 'task.json').read_text())['id'], 'payment-boundary')
        self.assertEqual(record, json.loads((self.out / 'preparation.json').read_text()))
    def test_go_requires_no_kiro_and_records_no_scaffold_claim(self):
        record = prepare_pilot.prepare(self.out, 'go', 'payment-boundary')
        self.assertEqual(record['toolchain'], {'go_version': 'go version go1.test'})
        self.assertFalse((self.out / 'service/.kiro').exists())
        self.assertIn('go vet ./...', (self.out / 'PROMPT.md').read_text())
        self.assertEqual(record['workspace_sha256'], prepare_pilot.hashes(self.out / 'service'))
        self.assertEqual(record['model_trials_run'], 0)
    def test_preparation_never_overwrites_existing_destination(self):
        self.out.mkdir(); sentinel = self.out / 'keep'; sentinel.write_text('preserve')
        with self.assertRaisesRegex(ValueError, 'already exists'):
            prepare_pilot.prepare(self.out, 'go', 'payment-boundary')
        self.assertEqual(sentinel.read_text(), 'preserve')
    def test_fixture_cannot_replace_scaffold_instructions(self):
        (self.suite / 'workload/AGENTS.md').write_text('unexpected replacement')
        with self.assertRaises((ValueError, AssertionError)):
            prepare_pilot.prepare(self.out, 'kiro', 'payment-boundary', self.binary)
        self.assertTrue((self.out / 'PREPARATION_FAILED').exists())
        self.assertFalse((self.out / 'preparation.json').exists())

    def test_missing_kiro_does_not_create_partial_workspace(self):
        with self.assertRaisesRegex(ValueError, '--kiro'):
            prepare_pilot.prepare(self.out, 'kiro', 'payment-boundary')
        self.assertFalse(self.out.exists())


if __name__ == '__main__': unittest.main()
