import json
from pathlib import Path
import subprocess
import tempfile
import unittest

import publish


class PublicationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / 'repo'
        self.repo.mkdir()
        self.git('init', '-b', 'master')
        self.git('config', 'user.name', 'Publication test')
        self.git('config', 'user.email', 'test@example.com')
        (self.repo / 'source.txt').write_text('before architecture')
        self.commit('Before Groma')
        architecture = self.repo / '.groma'
        architecture.mkdir()
        (architecture / 'project.md').write_text('Project')
        self.first = self.commit('First map')
        (self.repo / 'source.txt').write_text('new source')
        self.latest = self.commit('Description </script><img src=x onerror=alert(1)>')
        self.output = self.root / 'site'
        self.exporter = self.root / 'groma'
        self.exporter.write_text('''#!/usr/bin/env python3
import json, pathlib, sys
output = pathlib.Path(sys.argv[2])
output.mkdir(parents=True)
revision = sys.argv[sys.argv.index('--revision') + 1]
data = {'project': {'title': 'GHA', 'description': 'Short description',
                    'overview': 'Full overview\\n\\nFinal paragraph <script>unsafe</script>'},
        'revision': {'id': revision}}
(output / 'snapshot.js').write_text('globalThis.dispatchEvent(new CustomEvent("groma:published", { detail: '
                                  + json.dumps(data) + ' }));\\n')
(output / 'index.html').write_text(revision)
''')
        self.exporter.chmod(0o755)

    def git(self, *args):
        return subprocess.check_output(['git', '-C', str(self.repo), *args], text=True).strip()

    def commit(self, subject):
        self.git('add', '.')
        self.git('commit', '-m', subject)
        return self.git('rev-parse', 'HEAD')

    def build(self, limit=30):
        return publish.build(self.repo, self.output, 'https://example.com/gha/',
                             revision='HEAD', limit=limit, groma=str(self.exporter))

    def test_exports_history_newest_first_and_excludes_pre_groma_commits(self):
        revisions = self.build()
        self.assertEqual([item['id'] for item in revisions], [self.latest, self.first])
        for item in revisions:
            self.assertEqual((self.output / 'revisions' / item['id'] / 'index.html').read_text(),
                             item['id'])
        self.assertEqual(revisions[0]['overview'], 'Full overview\n\nFinal paragraph <script>unsafe</script>')

    def test_history_limit_is_bounded(self):
        revisions = self.build(limit=1)
        self.assertEqual([item['id'] for item in revisions], [self.latest])

    def test_first_parent_history_does_not_include_side_branch_commits(self):
        self.git('checkout', '-b', 'feature')
        (self.repo / 'feature.txt').write_text('feature')
        side = self.commit('Side branch change')
        self.git('checkout', 'master')
        self.git('merge', '--no-ff', 'feature', '-m', 'Merge feature')
        revisions = self.build()
        self.assertEqual(revisions[0]['subject'], 'Merge feature')
        self.assertNotIn(side, [item['id'] for item in revisions])

    def test_html_safely_preserves_subjects_and_complete_overview(self):
        self.build()
        page = (self.output / 'index.html').read_text()
        self.assertNotIn('</script><img', page)
        self.assertNotIn('<script>unsafe</script>', page)
        start = page.index('<script id="history-data" type="application/json">')
        data = page[start:].split('>', 1)[1].split('</script>', 1)[0]
        revisions = json.loads(data)
        self.assertIn('</script><img', revisions[0]['subject'])
        self.assertIn('Final paragraph', revisions[0]['overview'])
        self.assertIn('<details id="about">', page)
        self.assertIn('white-space:pre-wrap', page)
        self.assertIn('overflow:auto', page)

    def test_export_failure_never_writes_a_successful_site_index(self):
        self.exporter.write_text('#!/bin/sh\nexit 17\n')
        with self.assertRaises(subprocess.CalledProcessError):
            self.build()
        self.assertFalse((self.output / 'index.html').exists())

    def test_rejects_history_without_architecture(self):
        self.git('checkout', self.git('rev-list', '--max-parents=0', 'HEAD'))
        with self.assertRaisesRegex(ValueError, 'no committed Groma architecture'):
            self.build()

    def test_rejects_nonpositive_limit(self):
        with self.assertRaisesRegex(ValueError, 'positive'):
            self.build(limit=0)

    def test_refuses_to_mix_new_exports_into_an_existing_site(self):
        self.output.mkdir()
        (self.output / 'index.html').write_text('Previous deployment')
        with self.assertRaisesRegex(ValueError, 'already exists'):
            self.build()
        self.assertEqual((self.output / 'index.html').read_text(), 'Previous deployment')

    def test_unrecognized_export_metadata_stops_publication(self):
        self.exporter.write_text('''#!/usr/bin/env python3
import pathlib, sys
output = pathlib.Path(sys.argv[2])
output.mkdir(parents=True)
(output / 'snapshot.js').write_text('changed upstream format')
''')
        with self.assertRaisesRegex(ValueError, 'unsupported Groma snapshot envelope'):
            self.build()
        self.assertFalse((self.output / 'index.html').exists())

    def test_latest_revision_without_architecture_does_not_publish_an_old_map_as_latest(self):
        self.git('rm', '.groma/project.md')
        self.commit('Remove architecture')
        with self.assertRaisesRegex(ValueError, 'no committed Groma architecture'):
            self.build()


if __name__ == '__main__':
    unittest.main()
