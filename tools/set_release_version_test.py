import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location('release_version', Path(__file__).with_name('set-release-version.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class ReleaseVersionTests(unittest.TestCase):
    def test_tag_stamps_all_embedded_and_package_versions(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'frontend').mkdir()
            (root / 'wails.json').write_text(json.dumps({'info': {'productVersion': '1.0.0'}}))
            (root / 'frontend/package.json').write_text(json.dumps({'version': '1.0.0'}))
            (root / 'frontend/package-lock.json').write_text(json.dumps({'version': '1.0.0', 'packages': {'': {'version': '1.0.0'}, 'node_modules/example': {'version': '8.0.0'}}}))
            self.assertEqual(module.stamp(root, 'v1.2.3'), '1.2.3')
            self.assertEqual(json.loads((root / 'wails.json').read_text())['info']['productVersion'], '1.2.3')
            self.assertEqual(json.loads((root / 'frontend/package.json').read_text())['version'], '1.2.3')
            lock = json.loads((root / 'frontend/package-lock.json').read_text())
            self.assertEqual(lock['version'], '1.2.3')
            self.assertEqual(lock['packages']['']['version'], '1.2.3')
            self.assertEqual(lock['packages']['node_modules/example']['version'], '8.0.0')
    def test_invalid_tag_rejected_before_touching_files(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(ValueError):
                module.stamp(Path(directory), 'v1.2.3;bad')
