"""Tests for the installer packager.

The .deb path is exercised for real when dpkg-deb is available, because the
package layout and control file are the parts a broken installer would fail at
before a user ever gets to run one.
"""
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import package_installers

ROOT = Path(__file__).resolve().parents[1]


def fake_root(tmp):
    root = Path(tmp)
    (root / 'bin').mkdir(parents=True)
    (root / 'bin' / package_installers.BINARY).write_bytes(b'#!/bin/sh\nexit 0\n')
    (root / 'build' / 'linux').mkdir(parents=True)
    shutil.copy2(ROOT / 'build' / 'appicon.png', root / 'build' / 'appicon.png')
    shutil.copy2(
        ROOT / 'build' / 'linux' / f'{package_installers.SLUG}.desktop',
        root / 'build' / 'linux' / f'{package_installers.SLUG}.desktop',
    )
    return root


class ControlFileTest(unittest.TestCase):
    def test_names_the_package_version_and_runtime_dependencies(self):
        control = package_installers.deb_control('2.1.0', 512)
        self.assertIn('Package: guildsync-voice-mute', control)
        self.assertIn('Version: 2.1.0', control)
        self.assertIn('Depends: libgtk-4-1, libwebkitgtk-6.0-4', control)
        self.assertIn('Installed-Size: 512', control)

    def test_description_continuation_lines_are_indented(self):
        lines = package_installers.deb_control('1.0.0', 10).splitlines()
        description = lines.index('Description: GuildSync voice-channel mute companion')
        for line in lines[description + 1:]:
            self.assertTrue(line.startswith(' '), f'not a continuation line: {line!r}')


class DesktopEntryTest(unittest.TestCase):
    def test_points_at_the_installed_binary_and_icon(self):
        text = (ROOT / 'build' / 'linux' / f'{package_installers.SLUG}.desktop').read_text()
        self.assertIn('Exec=guildsync-voice-mute', text)
        self.assertIn('Icon=guildsync-voice-mute', text)
        self.assertIn('Type=Application', text)
        self.assertIn('Categories=', text)


class InstallerErrorsTest(unittest.TestCase):
    def test_missing_payload_is_reported_before_any_tool_runs(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(ValueError):
                package_installers.linux_package(tmp, '1.0.0')
            with self.assertRaises(ValueError):
                package_installers.macos_installer(tmp, '1.0.0')
            with self.assertRaises(ValueError):
                package_installers.windows_installer(tmp, '1.0.0', iscc=sys.executable)


@unittest.skipUnless(shutil.which('dpkg-deb'), 'dpkg-deb is not installed')
class DebBuildTest(unittest.TestCase):
    def test_deb_installs_the_binary_desktop_entry_and_icon(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = fake_root(tmp)
            archive = package_installers.linux_package(root, '2.1.0')
            self.assertEqual(archive.name, 'GuildSync-Voice-Mute-2.1.0-Linux.deb')

            contents = subprocess.run(
                ['dpkg-deb', '--contents', str(archive)],
                check=True, capture_output=True, text=True,
            ).stdout
            self.assertIn('./usr/bin/guildsync-voice-mute', contents)
            self.assertIn('./usr/share/applications/guildsync-voice-mute.desktop', contents)
            self.assertIn('./usr/share/icons/hicolor/256x256/apps/guildsync-voice-mute.png', contents)

            fields = subprocess.run(
                ['dpkg-deb', '--field', str(archive)],
                check=True, capture_output=True, text=True,
            ).stdout
            self.assertIn('Package: guildsync-voice-mute', fields)
            self.assertIn('Version: 2.1.0', fields)
            self.assertIn('Architecture: amd64', fields)
            self.assertIn('Depends: libgtk-4-1, libwebkitgtk-6.0-4', fields)


if __name__ == '__main__':
    unittest.main()
