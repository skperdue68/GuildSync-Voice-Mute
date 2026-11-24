"""Tests for the apt mirror repair.

The repair exists because a runner whose archive mirror crawls turns a 20-second
apt install into a 20-minute one, so these check the rewrite itself and that no
job installs native dependencies without running it first.
"""
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import apt_mirrors

ROOT = Path(__file__).resolve().parents[1]


def runner_root(tmp):
    """The apt layout a GitHub Ubuntu runner ships."""
    root = Path(tmp) / 'apt'
    (root / 'sources.list.d').mkdir(parents=True)
    (root / 'apt-mirrors.txt').write_text(
        'azure.archive.ubuntu.com/ubuntu/\tpriority:1\n'
        'archive.ubuntu.com/ubuntu/\tpriority:2\n'
        'security.ubuntu.com/ubuntu/\tpriority:3\n'
    )
    (root / 'sources.list.d' / 'ubuntu.sources').write_text(
        'Types: deb\n'
        'URIs: mirror+file:/etc/apt/apt-mirrors.txt\n'
        'Suites: noble noble-updates\n'
    )
    return root


class MirrorListTest(unittest.TestCase):
    def test_drops_the_azure_mirror_and_keeps_the_rest_ranked(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp)
            changes = apt_mirrors.rewrite(root)
            text = (root / 'apt-mirrors.txt').read_text()
            self.assertEqual(len(changes), 1)
            self.assertNotIn(apt_mirrors.AZURE, text)
            self.assertLess(text.index(apt_mirrors.ARCHIVE), text.index('security.ubuntu.com'))

    def test_a_list_that_only_holds_the_azure_mirror_falls_back(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp)
            (root / 'apt-mirrors.txt').write_text('azure.archive.ubuntu.com/ubuntu/\tpriority:1\n')
            apt_mirrors.rewrite(root)
            text = (root / 'apt-mirrors.txt').read_text()
            self.assertIn(apt_mirrors.ARCHIVE, text)
            self.assertIn('security.ubuntu.com', text)

    def test_an_already_repaired_tree_is_left_alone(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp)
            apt_mirrors.rewrite(root)
            self.assertEqual(apt_mirrors.rewrite(root), [])

    def test_an_image_without_the_mirror_file_is_not_a_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'apt'
            root.mkdir()
            self.assertEqual(apt_mirrors.rewrite(root), [])


class SourcesTest(unittest.TestCase):
    def test_a_source_that_names_the_azure_mirror_directly_is_repaired(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp)
            sources = root / 'sources.list.d' / 'ubuntu.sources'
            sources.write_text('Types: deb\nURIs: http://azure.archive.ubuntu.com/ubuntu/\n')
            apt_mirrors.rewrite(root)
            text = sources.read_text()
            self.assertNotIn(apt_mirrors.AZURE, text)
            self.assertIn('http://archive.ubuntu.com/ubuntu/', text)


class WorkflowTest(unittest.TestCase):
    """A crawling mirror must not be able to stall a job that installs packages."""

    def test_every_apt_install_is_preceded_by_the_repair(self):
        for name in ('build.yml', 'codeql.yml'):
            text = (ROOT / '.github' / 'workflows' / name).read_text()
            self.assertLess(text.index('tools/apt_mirrors.py'), text.index('apt-get update'), name)
            self.assertEqual(
                text.count('apt-get update'),
                text.count('tools/apt_mirrors.py'),
                f'{name}: every apt-get install needs the mirror repair',
            )
