import json
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import metadata
import package_release

CONFIG = """version: '3'

info:
  productName: "GuildSync Voice Mute"
  productIdentifier: "me.perdues.guildsync.voicemute"
  version: "1.0.0" # The application version

dev_mode:
  root_path: .
"""

PLIST = """<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
    <dict>
        <key>CFBundleName</key>
            <string>GuildSync Voice Mute</string>
        <key>CFBundleIdentifier</key>
            <string>me.perdues.guildsync.voicemute</string>
        <key>CFBundleVersion</key>
            <string>1.0.0</string>
        <key>CFBundleShortVersionString</key>
            <string>1.0.0</string>
    </dict>
</plist>
"""

WINDOWS = {
    'fixed': {'file_version': '1.0.0', 'product_version': '1.0.0'},
    'info': {'0409': {'FileVersion': '1.0.0', 'ProductVersion': '1.0.0', 'ProductName': 'GuildSync Voice Mute'}},
}


def make_tree(root, version='1.0.0'):
    (root / 'build/darwin').mkdir(parents=True)
    (root / 'build/windows').mkdir()
    (root / 'frontend').mkdir()
    (root / 'docs').mkdir()
    (root / 'bin').mkdir()
    (root / 'build/config.yml').write_text(CONFIG.replace('1.0.0', version))
    (root / 'build/darwin/Info.plist').write_text(PLIST.replace('1.0.0', version))
    (root / 'build/darwin/Info.dev.plist').write_text(PLIST.replace('1.0.0', version))
    (root / 'build/windows/info.json').write_text(json.dumps(WINDOWS).replace('1.0.0', version))
    (root / 'frontend/package.json').write_text(json.dumps({'version': version}))
    (root / 'frontend/package-lock.json').write_text(json.dumps({'version': version, 'packages': {'': {'version': version}}}))
    (root / 'README.md').write_text('# readme')
    (root / 'docs/platform-support.md').write_text('# platforms')
    (root / 'bin/app').write_text('binary')


class MetadataTests(unittest.TestCase):
    def test_consistent_tree_validates_and_accepts_matching_tag(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            make_tree(root)
            self.assertEqual(metadata.validate(root), '1.0.0')
            self.assertEqual(metadata.validate(root, 'v1.0.0'), '1.0.0')

    def test_mismatched_plist_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            make_tree(root)
            (root / 'build/darwin/Info.plist').write_text(PLIST.replace('1.0.0', '0.9.0'))
            with self.assertRaises(ValueError):
                metadata.validate(root)

    def test_wrong_product_identity_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            make_tree(root)
            (root / 'build/windows/info.json').write_text(json.dumps(WINDOWS).replace('GuildSync Voice Mute', 'My Product'))
            with self.assertRaises(ValueError):
                metadata.validate(root)

    def test_tag_mismatch_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            make_tree(root)
            with self.assertRaises(ValueError):
                metadata.validate(root, 'v1.2.3')

    def test_package_archives_bin_and_docs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            make_tree(root)
            archive = package_release.package(root, 'Linux', 'v1.0.0')
            self.assertEqual(archive.name, 'GuildSync-Voice-Mute-1.0.0-Linux.zip')
            with zipfile.ZipFile(archive) as bundle:
                self.assertEqual(sorted(bundle.namelist()), ['README.md', 'app', 'docs/platform-support.md'])

    def test_package_refuses_empty_build(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            make_tree(root)
            (root / 'bin/app').unlink()
            with self.assertRaises(ValueError):
                package_release.package(root, 'Linux')


if __name__ == '__main__':
    unittest.main()
