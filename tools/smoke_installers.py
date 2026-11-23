"""Prove the Windows and macOS packagers run on their own platforms.

The pull-request gate builds no application, so the installer paths are only
exercised on a release. This script builds a throwaway payload with the shape a
packager reads, runs the packager for the host platform, and checks what came
out. Run it as `python tools/smoke_installers.py windows` on a Windows runner
and `... macos` on a macOS runner.
"""
import argparse
import shutil
import subprocess
import sys
import tempfile
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
        builder = package_installers.windows_installer if arguments.platform == 'windows' else package_installers.macos_installer
        archive = builder(root, SMOKE_VERSION)
        size = archive.stat().st_size
        # An empty or tiny file means Inno Setup or hdiutil did not really run.
        if size < 100_000:
            raise SystemExit(f'::error::{archive.name} is only {size} bytes')
        print(f'{archive.name}: {size} bytes')
        if arguments.platform == 'windows':
            verify_windows(archive)
        else:
            verify_macos(archive)
    return 0


if __name__ == '__main__':
    sys.exit(main())
