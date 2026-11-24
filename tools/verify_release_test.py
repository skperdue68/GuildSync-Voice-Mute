import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import verify_release


class FindInstallerTest(unittest.TestCase):
    def test_picks_the_installer_beside_the_other_release_files(self):
        files = {
            'windows': (
                'GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe',
                ['GuildSync-Voice-Mute-2.1.0-Windows.zip', 'SHA256SUMS'],
            ),
            'macos': (
                'GuildSync-Voice-Mute-2.1.0-macOS.dmg',
                ['GuildSync-Voice-Mute-2.1.0-macOS.zip', 'SHA256SUMS'],
            ),
            'linux': (
                'GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage',
                ['GuildSync-Voice-Mute-2.1.0-Linux.zip', 'GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage.zip', 'SHA256SUMS'],
            ),
        }
        for platform, (installer, others) in files.items():
            with tempfile.TemporaryDirectory() as directory:
                for name in (installer, *others):
                    (Path(directory) / name).write_bytes(b'payload')
                found = verify_release.find_installer(directory, platform)
                self.assertEqual(found.name, installer)

    def test_refuses_a_missing_installer(self):
        with tempfile.TemporaryDirectory() as directory:
            (Path(directory) / 'GuildSync-Voice-Mute-2.1.0-Windows.zip').write_bytes(b'payload')
            with self.assertRaises(ValueError):
                verify_release.find_installer(directory, 'windows')

    def test_refuses_an_ambiguous_installer(self):
        with tempfile.TemporaryDirectory() as directory:
            for name in ('GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage',
                         'GuildSync-Voice-Mute-2.1.0-Linux-aarch64.AppImage'):
                (Path(directory) / name).write_bytes(b'payload')
            with self.assertRaises(ValueError):
                verify_release.find_installer(directory, 'linux')


if __name__ == '__main__':
    unittest.main()
