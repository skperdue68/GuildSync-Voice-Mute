"""Prove the Windows and macOS packagers and their published zips on their own platforms.

The pull-request gate builds no application, so the installer paths are only
exercised on a release. This script builds a throwaway payload with the shape a
packager reads, packs it through the same build-installer-and-zip step a release
runs, and then installs from the zip that step published. Run it as
`python tools/smoke_installers.py windows` on a Windows runner and
`... macos` on a macOS runner.
"""
import argparse
import shutil
import subprocess
import sys
import tempfile
import zipfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import package_installers

SOURCE_ROOT = Path(__file__).resolve().parents[1]
SMOKE_VERSION = '0.0.1'


def stub_root(destination, platform):
    """Create only the files one packager reads, with a stub payload."""
    root = Path(destination)
    payload = root / 'bin'
    payload.mkdir(parents=True)
    if platform == 'windows':
        (payload / f'{package_installers.BINARY}.exe').write_bytes(b'MZ stub payload, not a real executable\n')
        script = root / 'build' / 'windows' / 'installer.iss'
        script.parent.mkdir(parents=True)
        shutil.copy2(SOURCE_ROOT / 'build' / 'windows' / 'installer.iss', script)
    else:
        executable = payload / f'{package_installers.BINARY}.app' / 'Contents' / 'MacOS' / package_installers.BINARY
        executable.parent.mkdir(parents=True)
        executable.write_bytes(b'#!/bin/sh\nexit 0\n')
        executable.chmod(0o755)
    return root


def package(root, platform):
    """Build the installer and the zip a release publishes from a stub payload."""
    return package_installers.build_installer(root, platform, SMOKE_VERSION)


def unpack(download, destination):
    """Extract the published zip and return the one installer it holds.

    The update hands the user this zip, so the copy that gets verified is the
    one a user ends up with, not the installer the packager left behind.
    """
    with zipfile.ZipFile(download) as bundle:
        names = bundle.namelist()
        if len(names) != 1:
            raise SystemExit(f'::error::{download.name} holds {names}, not one installer')
        return Path(bundle.extract(names[0], destination))


def verify_windows(archive):
    """Run the setup silently and confirm it installs the application."""
    with tempfile.TemporaryDirectory() as install:
        target = Path(install) / 'installed'
        # No /CURRENTUSER: the script already installs per user and never
        # elevates, and Inno only accepts that flag when the script allows it.
        subprocess.run(
            [str(archive), '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', f'/DIR={target}'],
            check=True,
            timeout=600,
        )
        installed = target / f'{package_installers.BINARY}.exe'
        if not installed.is_file():
            raise SystemExit(f'::error::the installer did not put {installed.name} in {target}')
        print(f'installed {installed}')
        uninstaller = target / 'unins000.exe'
        if uninstaller.is_file():
            subprocess.run([str(uninstaller), '/VERYSILENT', '/SUPPRESSMSGBOXES'], check=False, timeout=600)


def verify_macos(archive):
    """Mount the DMG and confirm the bundle and the Applications link are in it."""
    with tempfile.TemporaryDirectory() as mount:
        subprocess.run(
            ['hdiutil', 'attach', str(archive), '-nobrowse', '-readonly', '-mountpoint', mount],
            check=True,
            capture_output=True,
        )
        try:
            executable = Path(mount) / f'{package_installers.BINARY}.app' / 'Contents' / 'MacOS' / package_installers.BINARY
            if not executable.is_file():
                raise SystemExit(f'::error::the DMG has no {executable.relative_to(mount)}')
            if not (Path(mount) / 'Applications').is_symlink():
                raise SystemExit('::error::the DMG has no Applications link')
            print(f'mounted {executable}')
        finally:
            subprocess.run(['hdiutil', 'detach', mount, '-force'], check=False, capture_output=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('platform', choices=('windows', 'macos'))
    arguments = parser.parse_args()
    with tempfile.TemporaryDirectory() as workspace:
        root = stub_root(workspace, arguments.platform)
        archive, download = package(root, arguments.platform)
        size = archive.stat().st_size
        # Size proves nothing about a stub payload, so only an empty file is
        # rejected here; the checks below are what actually prove the installer.
        if size == 0:
            raise SystemExit(f'::error::{archive.name} is empty')
        print(f'{archive.name}: {size} bytes')
        unpacked = unpack(download, Path(workspace) / 'unpacked')
        if unpacked.name != archive.name:
            raise SystemExit(f'::error::{download.name} holds {unpacked.name}, not {archive.name}')
        if unpacked.read_bytes() != archive.read_bytes():
            raise SystemExit(f'::error::{unpacked.name} in {download.name} is not the built installer')
        print(f'{download.name}: {download.stat().st_size} bytes, holding {unpacked.name}')
        if arguments.platform == 'windows':
            verify_windows(unpacked)
        else:
            verify_macos(unpacked)
    return 0


if __name__ == '__main__':
    sys.exit(main())
