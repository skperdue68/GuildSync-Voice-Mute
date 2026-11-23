"""Prove each platform's installer on its own platform.

The pull-request gate builds no application, so the installer paths are only
exercised on a release. This script builds the payload each packager reads,
packs it through the same build-installer step a release runs, and then verifies
the installer that step published. Windows and macOS package a stub payload,
because their packagers only need a file of the right shape; Linux packages the
application the job just built, because the AppImage bundler reads the binary's
linked libraries. Run it as `python tools/smoke_installers.py windows` on a
Windows runner, `... macos` on a macOS runner and `... linux` on an Ubuntu
runner with GTK4/WebKitGTK installed.
"""
import argparse
import os
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


def payload_root(workspace, platform):
    """The root a packager reads for this platform.

    The AppImage bundler runs ldd on the binary it is given and fails on a stub,
    so Linux packages the build this job made rather than a fake payload.
    """
    if platform == 'linux':
        return SOURCE_ROOT
    return stub_root(workspace, platform)


def package(root, platform):
    """Build the installer a release publishes."""
    if platform == 'linux':
        # The generator names the AppImage after the architecture it built.
        return package_installers.build_installer(root, platform, SMOKE_VERSION, os.uname().machine)
    return package_installers.build_installer(root, platform, SMOKE_VERSION)


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


def verify_linux(archive):
    """Extract the AppImage and confirm it carries the application and its launcher."""
    with tempfile.TemporaryDirectory() as extraction:
        subprocess.run(
            [str(archive), '--appimage-extract'],
            cwd=extraction,
            check=True,
            capture_output=True,
            timeout=900,
        )
        root = Path(extraction) / 'squashfs-root'
        if not (root / 'AppRun').is_file():
            raise SystemExit('::error::the AppImage has no AppRun')
        binary = root / 'usr' / 'bin' / package_installers.SLUG
        if not binary.is_file():
            raise SystemExit(f'::error::the AppImage has no usr/bin/{package_installers.SLUG}')
        entry = root / f'{package_installers.SLUG}.desktop'
        installed = root / 'usr' / 'share' / 'applications' / entry.name
        if not entry.is_file() and not installed.is_file():
            raise SystemExit(f'::error::the AppImage has no {entry.name}')
        if not any(root.rglob(f'{package_installers.SLUG}.png')):
            raise SystemExit('::error::the AppImage has no application icon')
        print(f'extracted {binary.relative_to(root)} and its launcher')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('platform', choices=('windows', 'macos', 'linux'))
    arguments = parser.parse_args()
    with tempfile.TemporaryDirectory() as workspace:
        root = payload_root(workspace, arguments.platform)
        archive = package(root, arguments.platform)
        size = archive.stat().st_size
        # Size proves nothing about a stub payload, so only an empty file is
        # rejected here; the checks below are what actually prove the installer.
        if size == 0:
            raise SystemExit(f'::error::{archive.name} is empty')
        print(f'{archive.name}: {size} bytes')
        if arguments.platform == 'windows':
            verify_windows(archive)
        elif arguments.platform == 'macos':
            verify_macos(archive)
        else:
            verify_linux(archive)
    return 0


if __name__ == '__main__':
    sys.exit(main())
