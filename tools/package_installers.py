"""Build the platform installers for a release.

The release zips stay portable archives. These are the installers users expect
next to them: an Inno Setup .exe on Windows, a drag-to-Applications .dmg on
macOS and an AppImage on Linux, which carries the GTK/WebKitGTK runtime it
links against and so runs on any distribution. All three package the same bin/
payload the zip uses, refuse to run when the product metadata is inconsistent,
and publish the finished installer inside a zip as well, because that zip is
the file the in-app update downloads.
"""
import argparse
import os
import shutil
import subprocess
import sys
import tempfile
import zipfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import metadata

BINARY = 'GuildSync-Voice-Mute'
SLUG = 'guildsync-voice-mute'
# wails3's AppImage generator builds these two architectures and names the file
# after the one it built. The name is published, so it is checked.
APPIMAGE_ARCHITECTURES = ('x86_64', 'aarch64')
ISCC_CANDIDATES = (
    r'C:\Program Files (x86)\Inno Setup 6\ISCC.exe',
    r'C:\Program Files\Inno Setup 6\ISCC.exe',
)


def release_dir(root):
    directory = Path(root) / 'release'
    directory.mkdir(exist_ok=True)
    return directory


def zip_installer(archive):
    """Publish the installer inside a zip and return the zip's path.

    The updater downloads this zip rather than the bare installer, so what a run
    of the app fetches is one archive format on every platform, named after the
    installer it holds.
    """
    archive = Path(archive)
    download = archive.with_name(archive.name + '.zip')
    with zipfile.ZipFile(download, 'w', zipfile.ZIP_DEFLATED) as bundle:
        bundle.write(archive, archive.name)
    return download


def build_installer(root, platform, version, architecture='x86_64', iscc=None):
    """Build one platform's installer and publish it inside its zip.

    A release runs exactly this, so the archive a smoke test builds is the one
    the updater downloads rather than a bare installer shaped like it.
    """
    if platform == 'linux':
        archive = linux_package(root, version, architecture)
    elif platform == 'macos':
        archive = macos_installer(root, version)
    elif platform == 'windows':
        archive = windows_installer(root, version, iscc)
    else:
        raise ValueError(f'{platform!r} is not a platform installer')
    return archive, zip_installer(archive)


def appimage_command(binary, icon, desktop, output, build):
    """The wails3 invocation that bundles one AppImage."""
    return [
        'wails3', 'generate', 'appimage',
        '-binary', str(binary),
        '-icon', str(icon),
        '-desktopfile', str(desktop),
        '-outputdir', str(output),
        '-builddir', str(build),
    ]


def linux_package(root, version, architecture='x86_64', run=subprocess.run):
    """Build an AppImage for one architecture.

    The AppImage is the Linux installer because it bundles the GTK/WebKitGTK
    runtime it links against, so one file runs on any distribution rather than
    only on the one whose package manager would accept a .deb. It is still
    built for a single architecture, so the architecture is part of the
    filename: the updater picks the one built for the machine it runs on.
    """
    if architecture not in APPIMAGE_ARCHITECTURES:
        raise ValueError(f'{architecture!r} is not an AppImage architecture')
    root = Path(root)
    payload = root / 'bin' / BINARY
    if not payload.is_file():
        raise ValueError(f'the Linux payload is missing: {payload}')
    if run is subprocess.run and not shutil.which('wails3'):
        raise ValueError('the wails3 CLI is required to build the AppImage')
    with tempfile.TemporaryDirectory() as staging:
        stage = Path(staging)
        # The generator names the AppImage after the binary and writes an AppRun
        # that runs that name, so the payload is staged under the name the
        # desktop file's Exec line already uses.
        binary = stage / SLUG
        shutil.copy2(payload, binary)
        binary.chmod(0o755)
        # linuxdeploy looks the icon up by the name the desktop file's Icon= line
        # uses, so it is staged under that name too.
        icon = stage / f'{SLUG}.png'
        shutil.copy2(root / 'build' / 'appicon.png', icon)
        output = release_dir(root)
        run(appimage_command(
            binary,
            icon,
            root / 'build' / 'linux' / f'{SLUG}.desktop',
            output,
            stage / 'appimage',
        ), check=True)
        produced = output / f'{SLUG}-{architecture}.AppImage'
        if not produced.is_file():
            raise ValueError(f'wails3 reported success but {produced.name} was not written')
        archive = output / f'{BINARY}-{version}-Linux-{architecture}.AppImage'
        produced.replace(archive)
        archive.chmod(0o755)
        return archive


def macos_installer(root, version):
    """Build a .dmg holding the .app bundle and an Applications link."""
    root = Path(root)
    bundles = sorted((root / 'bin').glob('*.app'))
    if not bundles:
        raise ValueError('the macOS payload is missing: no .app bundle in bin/')
    bundle = bundles[0]
    # A bundle without its executable would produce a DMG that installs a
    # broken app, so check the payload before spending a runner building one.
    executables = bundle / 'Contents' / 'MacOS'
    if not executables.is_dir() or not any(path.is_file() for path in executables.iterdir()):
        raise ValueError(f'{bundle.name} has no executable in Contents/MacOS')
    if not shutil.which('hdiutil'):
        raise ValueError('a .dmg can only be built on macOS (hdiutil is missing)')
    with tempfile.TemporaryDirectory() as staging:
        stage = Path(staging)
        shutil.copytree(bundle, stage / bundle.name, symlinks=True)
        (stage / 'Applications').symlink_to('/Applications')
        archive = release_dir(root) / f'{BINARY}-{version}-macOS.dmg'
        subprocess.run(
            ['hdiutil', 'create', '-volname', 'GuildSync Voice Mute', '-srcfolder', str(stage),
             '-ov', '-format', 'UDZO', str(archive)],
            check=True,
        )
        return archive


def find_iscc():
    candidates = (os.environ.get('INNO_SETUP_ISCC'), shutil.which('ISCC.exe'), shutil.which('iscc'), *ISCC_CANDIDATES)
    for candidate in candidates:
        if candidate and Path(candidate).is_file():
            return Path(candidate)
    raise ValueError('Inno Setup is required to build the Windows installer (ISCC.exe was not found)')


def windows_installer(root, version, iscc=None):
    """Build the Inno Setup installer from build/windows/installer.iss."""
    root = Path(root)
    executable = root / 'bin' / f'{BINARY}.exe'
    if not executable.is_file():
        raise ValueError(f'the Windows payload is missing: {executable}')
    tool = Path(iscc) if iscc else find_iscc()
    output = release_dir(root)
    subprocess.run(
        [str(tool), f'/DMyAppVersion={version}', f'/DSourceRoot={root / "bin"}',
         f'/DOutputRoot={output}', str(root / 'build' / 'windows' / 'installer.iss')],
        check=True,
    )
    archive = output / f'{BINARY}-{version}-Windows-Setup.exe'
    if not archive.is_file():
        raise ValueError(f'Inno Setup reported success but {archive} was not written')
    return archive


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('platform', choices=('linux', 'macos', 'windows'))
    parser.add_argument('--version', help='Release tag the metadata must match')
    parser.add_argument('--arch', default='x86_64', help='AppImage architecture for the Linux installer')
    parser.add_argument('--iscc', help='Path to Inno Setup ISCC.exe')
    arguments = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    try:
        version = metadata.validate(root, arguments.version)
        archive, download = build_installer(
            root, arguments.platform, version, arguments.arch, arguments.iscc
        )
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f'::error::installer packaging failed: {error}')
        return 1
    print(archive)
    print(download)
    return 0


if __name__ == '__main__':
    sys.exit(main())
