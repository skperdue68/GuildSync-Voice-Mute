"""Tests for the installer packager.

The packagers are driven with a stand-in for the tool that does the final step,
because those tools only exist on the platform a release builds on. What is
checked here is the command each packager builds and the name it publishes,
which is what the updater matches at download time.
"""
import os
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import package_installers
import smoke_installers

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


class MacBundleTaskTest(unittest.TestCase):
    """The .app bundle is assembled by the darwin Taskfile, not by `go build`.

    A release failed with "no .app bundle in bin/" because the build wrote only
    a bare binary, so these assert the builder, the bundle metadata and the
    workflow still agree on where the DMG packager reads it from.
    """

    def test_taskfile_writes_the_bundle_the_packager_reads(self):
        text = (ROOT / 'build' / 'darwin' / 'Taskfile.yml').read_text()
        self.assertIn('bundle="{{.BIN_DIR}}/{{.APP_NAME}}.app"', text)
        self.assertIn('"$bundle/Contents/Info.plist"', text)
        self.assertIn('"$bundle/Contents/Resources/icons.icns"', text)
        self.assertIn('"$bundle/Contents/MacOS/{{.APP_NAME}}"', text)

    def test_bundle_executable_is_the_name_the_packager_requires(self):
        plist = (ROOT / 'build' / 'darwin' / 'Info.plist').read_text()
        executable = re.search(r'<key>CFBundleExecutable</key>\s*<string>([^<]+)</string>', plist)
        self.assertIsNotNone(executable)
        self.assertEqual(executable.group(1), package_installers.BINARY)

    def test_workflow_builds_the_bundle_before_signing_it(self):
        text = (ROOT / '.github' / 'workflows' / 'build.yml').read_text()
        self.assertIn('wails3 task darwin:package', text)
        self.assertLess(
            text.index('wails3 task darwin:package'),
            text.index('wails3 task darwin:codesign'),
        )


class DesktopEntryTest(unittest.TestCase):
    def test_points_at_the_installed_binary_and_icon(self):
        text = (ROOT / 'build' / 'linux' / f'{package_installers.SLUG}.desktop').read_text()
        self.assertIn('Exec=guildsync-voice-mute', text)
        self.assertIn('Icon=guildsync-voice-mute', text)
        self.assertIn('Type=Application', text)
        self.assertIn('Categories=', text)


class InstallerErrorsTest(unittest.TestCase):
    def test_an_unknown_platform_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(ValueError):
                package_installers.build_installer(tmp, 'linux-arm', '1.0.0')

    def test_missing_payload_is_reported_before_any_tool_runs(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(ValueError):
                package_installers.linux_package(tmp, '1.0.0')
            with self.assertRaises(ValueError):
                package_installers.macos_installer(tmp, '1.0.0')
            with self.assertRaises(ValueError):
                package_installers.windows_installer(tmp, '1.0.0', iscc=sys.executable)


class StubPayloadTest(unittest.TestCase):
    def test_windows_stub_carries_what_the_packager_reads(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = smoke_installers.stub_root(tmp, 'windows')
            self.assertTrue((root / 'bin' / f'{package_installers.BINARY}.exe').is_file())
            self.assertTrue((root / 'build' / 'windows' / 'installer.iss').is_file())

    def test_macos_stub_is_a_bundle_with_an_executable(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = smoke_installers.stub_root(tmp, 'macos')
            executable = root / 'bin' / f'{package_installers.BINARY}.app' / 'Contents' / 'MacOS' / package_installers.BINARY
            self.assertTrue(executable.is_file())
            self.assertTrue(os.access(executable, os.X_OK))


class AppImageBuildTest(unittest.TestCase):
    def test_publishes_an_appimage_named_for_the_architecture(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = fake_root(tmp)
            commands = []

            def fake_run(command, check):
                commands.append(command)
                # The generator writes the file it names after the binary.
                (root / 'release' / f'{package_installers.SLUG}-x86_64.AppImage').write_bytes(b'AppImage')
                return subprocess.CompletedProcess(command, 0)

            archive = package_installers.linux_package(root, '2.1.0', run=fake_run)
            self.assertEqual(archive.name, 'GuildSync-Voice-Mute-2.1.0-Linux-x86_64.AppImage')
            self.assertTrue(os.access(archive, os.X_OK))
            self.assertFalse((root / 'release' / f'{package_installers.SLUG}-x86_64.AppImage').exists())

            command = commands[0]
            self.assertEqual(command[:3], ['wails3', 'generate', 'appimage'])
            # The binary is staged under the name the desktop entry launches, so
            # the AppImage's own AppRun matches it.
            self.assertEqual(Path(command[command.index('-binary') + 1]).name, package_installers.SLUG)
            self.assertEqual(Path(command[command.index('-desktopfile') + 1]).name, f'{package_installers.SLUG}.desktop')
            # linuxdeploy resolves the icon by the desktop entry's Icon= name.
            self.assertEqual(Path(command[command.index('-icon') + 1]).name, f'{package_installers.SLUG}.png')
            self.assertEqual(Path(command[command.index('-outputdir') + 1]), root / 'release')

    def test_arm64_keeps_the_architecture_the_generator_names(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = fake_root(tmp)

            def fake_run(command, check):
                (root / 'release' / f'{package_installers.SLUG}-aarch64.AppImage').write_bytes(b'AppImage')

            archive = package_installers.linux_package(root, '2.1.0', 'aarch64', run=fake_run)
            self.assertEqual(archive.name, 'GuildSync-Voice-Mute-2.1.0-Linux-aarch64.AppImage')

    def test_a_missing_output_is_reported_instead_of_publishing_nothing(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = fake_root(tmp)
            with self.assertRaises(ValueError):
                package_installers.linux_package(root, '2.1.0', run=lambda command, check: None)


class InstallerZipTest(unittest.TestCase):
    def test_the_download_is_a_zip_of_the_installer(self):
        with tempfile.TemporaryDirectory() as tmp:
            archive = Path(tmp) / 'GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe'
            archive.write_bytes(b'setup payload')
            download = package_installers.zip_installer(archive)
            self.assertEqual(download.name, 'GuildSync-Voice-Mute-2.1.0-Windows-Setup.exe.zip')
            with zipfile.ZipFile(download) as bundle:
                self.assertEqual(bundle.namelist(), [archive.name])
                self.assertEqual(bundle.read(archive.name), b'setup payload')


class BuildInstallerTest(unittest.TestCase):
    def test_publishes_the_zip_of_the_installer_the_packager_wrote(self):
        with tempfile.TemporaryDirectory() as tmp:
            archive = Path(tmp) / 'GuildSync-Voice-Mute-1.0.0-Linux-x86_64.AppImage'
            archive.write_bytes(b'AppImage')
            with mock.patch.object(package_installers, 'linux_package', return_value=archive):
                built, download = package_installers.build_installer(Path(tmp), 'linux', '1.0.0')
            self.assertEqual(built, archive)
            self.assertEqual(download.name, archive.name + '.zip')
            with zipfile.ZipFile(download) as bundle:
                self.assertEqual(bundle.namelist(), [archive.name])


class SmokeZipTest(unittest.TestCase):
    """The pull-request smoke job proves the zip the updater downloads."""

    def test_the_smoke_packages_through_the_release_path(self):
        stub = Path('stub')
        with mock.patch.object(
            package_installers, 'build_installer', return_value=('installer', 'zip')
        ) as builder:
            self.assertEqual(smoke_installers.package(stub, 'macos'), ('installer', 'zip'))
        builder.assert_called_once_with(stub, 'macos', smoke_installers.SMOKE_VERSION)

    def test_the_installer_is_unpacked_for_verification(self):
        with tempfile.TemporaryDirectory() as tmp:
            archive = Path(tmp) / 'GuildSync-Voice-Mute-0.0.1-Windows-Setup.exe'
            archive.write_bytes(b'setup payload')
            unpacked = smoke_installers.unpack(package_installers.zip_installer(archive), Path(tmp) / 'unpacked')
            self.assertEqual(unpacked.name, archive.name)
            self.assertEqual(unpacked.read_bytes(), archive.read_bytes())

    def test_the_unpacked_installer_keeps_its_executable_bit(self):
        with tempfile.TemporaryDirectory() as tmp:
            archive = Path(tmp) / 'GuildSync-Voice-Mute-0.0.1-Linux-x86_64.AppImage'
            archive.write_bytes(b'AppImage')
            archive.chmod(0o755)
            unpacked = smoke_installers.unpack(package_installers.zip_installer(archive), Path(tmp) / 'unpacked')
            self.assertTrue(os.access(unpacked, os.X_OK))

    def test_a_zip_holding_more_than_the_installer_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            download = Path(tmp) / 'installer.zip'
            with zipfile.ZipFile(download, 'w') as bundle:
                bundle.writestr('installer', b'one')
                bundle.writestr('readme', b'two')
            with self.assertRaises(SystemExit):
                smoke_installers.unpack(download, Path(tmp) / 'unpacked')


class SmokePayloadTest(unittest.TestCase):
    def test_linux_packages_the_application_instead_of_a_stub(self):
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(smoke_installers.payload_root(tmp, 'linux'), smoke_installers.SOURCE_ROOT)
            self.assertEqual(smoke_installers.payload_root(tmp, 'macos'), Path(tmp))


@unittest.skipUnless(sys.platform.startswith('linux'), 'the AppImage runtime is Linux-only')
class LinuxSmokeTest(unittest.TestCase):
    """The Linux leg verifies the AppImage the updater downloads."""

    def image(self, tmp, body):
        image = Path(tmp) / 'GuildSync-Voice-Mute-0.0.1-Linux-x86_64.AppImage'
        image.write_text('#!/bin/sh\nset -e\n' + body)
        image.chmod(0o755)
        return image

    def test_the_extracted_image_carries_the_application(self):
        with tempfile.TemporaryDirectory() as tmp:
            image = self.image(tmp, (
                'mkdir -p squashfs-root/usr/bin\n'
                'touch squashfs-root/AppRun\n'
                f'touch squashfs-root/{package_installers.SLUG}.desktop\n'
                f'touch squashfs-root/{package_installers.SLUG}.png\n'
                f'cp "$0" squashfs-root/usr/bin/{package_installers.SLUG}\n'
            ))
            smoke_installers.verify_linux(image)

    def test_an_image_without_the_application_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            image = self.image(tmp, 'mkdir -p squashfs-root\ntouch squashfs-root/AppRun\n')
            with self.assertRaises(SystemExit):
                smoke_installers.verify_linux(image)


class ArchitectureNameTest(unittest.TestCase):
    def test_an_architecture_the_generator_does_not_build_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = fake_root(tmp)
            for architecture in ('amd64', 'x86_64/../../evil', 'AMD64', '', 'i686'):
                with self.assertRaises(ValueError):
                    package_installers.linux_package(root, '1.0.0', architecture)


if __name__ == '__main__':
    unittest.main()
