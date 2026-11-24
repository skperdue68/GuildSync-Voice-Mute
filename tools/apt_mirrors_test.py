"""Tests for the apt mirror repair.

The repair exists because a runner whose regional archive mirror crawls turns a
20-second apt install into a 14-23 minute one, so these check the rewrite itself
and that no job installs native dependencies without running it first.
"""
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import apt_mirrors

ROOT = Path(__file__).resolve().parents[1]


def runner_root(tmp, mirrors_text):
    """The apt layout a GitHub Ubuntu runner ships, with its own mirror list."""
    root = Path(tmp) / 'apt'
    (root / 'sources.list.d').mkdir(parents=True)
    (root / apt_mirrors.MIRROR_LIST).write_text(mirrors_text)
    (root / 'sources.list.d' / 'ubuntu.sources').write_text(
        'Types: deb\n'
        'URIs: mirror+file:/etc/apt/apt-mirrors.txt\n'
        'Suites: noble noble-updates\n'
    )
    return root


EC2_MIRRORS = (
    'us-east-2.ec2.archive.ubuntu.com/ubuntu/\tpriority:1\n'
    'archive.ubuntu.com/ubuntu/\tpriority:2\n'
    'security.ubuntu.com/ubuntu/\tpriority:3\n'
)


class MirrorListTest(unittest.TestCase):
    def test_the_regional_archive_mirror_is_repointed_and_not_duplicated(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp, EC2_MIRRORS)
            changes = apt_mirrors.rewrite(root)
            text = (root / apt_mirrors.MIRROR_LIST).read_text()
            self.assertEqual(len(changes), 1)
            self.assertNotIn('ec2.archive.ubuntu.com', text)
            self.assertEqual(text.count('archive.ubuntu.com/ubuntu/'), 1)
            self.assertIn('security.ubuntu.com', text)

    def test_the_azure_mirror_is_repointed(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp, 'azure.archive.ubuntu.com/ubuntu/\tpriority:1\n')
            apt_mirrors.rewrite(root)
            text = (root / apt_mirrors.MIRROR_LIST).read_text()
            self.assertNotIn('azure.archive.ubuntu.com', text)
            self.assertIn('archive.ubuntu.com/ubuntu/', text)

    def test_canonical_hosts_are_matched_only_with_a_region_in_front(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp, 'archive.ubuntu.com/ubuntu/\tpriority:1\n')
            self.assertEqual(apt_mirrors.rewrite(root), [])

    def test_an_already_repaired_tree_is_left_alone(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp, EC2_MIRRORS)
            apt_mirrors.rewrite(root)
            self.assertEqual(apt_mirrors.rewrite(root), [])

    def test_an_image_without_the_mirror_file_is_not_a_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'apt'
            root.mkdir()
            self.assertEqual(apt_mirrors.rewrite(root), [])


class SourcesTest(unittest.TestCase):
    def test_a_source_that_names_a_regional_mirror_directly_is_repointed(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp, EC2_MIRRORS)
            sources = root / 'sources.list.d' / 'ubuntu.sources'
            sources.write_text(
                'Types: deb\n'
                'URIs: http://us-east-2.ec2.archive.ubuntu.com/ubuntu/\n'
            )
            apt_mirrors.rewrite(root)
            text = sources.read_text()
            self.assertNotIn('ec2.archive.ubuntu.com', text)
            self.assertIn('http://archive.ubuntu.com/ubuntu/', text)

    def test_a_source_already_on_the_canonical_mirror_is_untouched(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = runner_root(tmp, 'archive.ubuntu.com/ubuntu/\tpriority:1\n')
            sources = root / 'sources.list.d' / 'ubuntu.sources'
            sources.write_text('Types: deb\nURIs: http://archive.ubuntu.com/ubuntu/\n')
            self.assertEqual(apt_mirrors.rewrite(root), [])


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
