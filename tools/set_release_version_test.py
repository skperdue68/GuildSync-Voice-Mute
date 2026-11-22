import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location('release_version', Path(__file__).with_name('set-release-version.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

CONFIG = "version: '3'\n\ninfo:\n  productName: \"GuildSync Voice Mute\"\n  version: \"1.0.0\"\n\ndev_mode:\n  root_path: .\n"

PLIST = """<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
    <dict>
        <key>CFBundleVersion</key>
            <string>1.0.0</string>
        <key>CFBundleShortVersionString</key>
            <string>1.0.0</string>
    </dict>
</plist>
"""

WINDOWS = {'fixed': {'file_version': '1.0.0', 'product_version': '1.0.0'},
           'info': {'0409': {'FileVersion': '1.0.0', 'ProductVersion': '1.0.0', 'ProductName': 'GuildSync Voice Mute'}}}

class ReleaseVersionTests(unittest.TestCase):
    def _fixture(self, root):
        (root / 'build/darwin').mkdir(parents=True)
        (root / 'build/windows').mkdir()
        (root / 'frontend').mkdir()
        (root / 'build/config.yml').write_text(CONFIG)
        (root / 'build/darwin/Info.plist').write_text(PLIST)
        (root / 'build/darwin/Info.dev.plist').write_text(PLIST)
        (root / 'build/windows/info.json').write_text(json.dumps(WINDOWS))
        (root / 'frontend/package.json').write_text(json.dumps({'version': '1.0.0'}))
        (root / 'frontend/package-lock.json').write_text(json.dumps({'version': '1.0.0', 'packages': {'': {'version': '1.0.0'}, 'node_modules/example': {'version': '8.0.0'}}}))

    def test_tag_stamps_all_embedded_and_package_versions(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self._fixture(root)
            self.assertEqual(module.stamp(root, 'v1.2.3'), '1.2.3')

            config = (root / 'build/config.yml').read_text()
            self.assertIn("version: '3'", config, 'schema version must not change')
            self.assertIn('  version: "1.2.3"', config)

            for name in ('Info.plist', 'Info.dev.plist'):
                plist = (root / 'build/darwin' / name).read_text()
                self.assertEqual(plist.count('<string>1.2.3</string>'), 2)
                self.assertNotIn('<string>1.0.0</string>', plist)

            windows = json.loads((root / 'build/windows/info.json').read_text())
            self.assertEqual(windows['fixed']['product_version'], '1.2.3')
            self.assertEqual(windows['info']['0409']['FileVersion'], '1.2.3')
            self.assertEqual(windows['info']['0409']['ProductName'], 'GuildSync Voice Mute')

            self.assertEqual(json.loads((root / 'frontend/package.json').read_text())['version'], '1.2.3')
            lock = json.loads((root / 'frontend/package-lock.json').read_text())
            self.assertEqual(lock['version'], '1.2.3')
            self.assertEqual(lock['packages']['']['version'], '1.2.3')
            self.assertEqual(lock['packages']['node_modules/example']['version'], '8.0.0')

    def test_invalid_tag_rejected_before_touching_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self._fixture(root)
            with self.assertRaises(ValueError):
                module.stamp(root, 'v1.2.3;bad')
            self.assertIn('  version: "1.0.0"', (root / 'build/config.yml').read_text())
