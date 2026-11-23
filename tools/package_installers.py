"""Build the platform installers for a release.

The release zips stay portable archives. These are the installers users expect
next to them: an Inno Setup .exe on Windows, a drag-to-Applications .dmg on
macOS and a .deb on Linux. All three package the same bin/ payload the zip
uses, and refuse to run when the product metadata is inconsistent.
"""
import argparse
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import metadata

BINARY = 'GuildSync-Voice-Mute'
SLUG = 'guildsync-voice-mute'
MAINTAINER = 'GuildSync <58951920+skperdue68@users.noreply.github.com>'
# Ubuntu 24.04 package names for the GTK4/WebKitGTK 6.0 runtime the build needs.
DEPENDS = 'libgtk-4-1, libwebkitgtk-6.0-4'
ISCC_CANDIDATES = (
    r'C:\Program Files (x86)\Inno Setup 6\ISCC.exe',
    r'C:\Program Files\Inno Setup 6\ISCC.exe',
)


def release_dir(root):
    directory = Path(root) / 'release'
    directory.mkdir(exist_ok=True)
    return directory


def deb_control(version, installed_kb, architecture='amd64'):
    """Render the DEBIAN/control file for the Linux package.

    Debian keeps a package description's continuation lines indented by one
    space; anything else is read as a new field and makes dpkg reject the file.
    """
    fields = [
        f'Package: {SLUG}',
        f'Version: {version}',
        'Section: net',
        'Priority: optional',
        f'Architecture: {architecture}',
        f'Maintainer: {MAINTAINER}',
        f'Installed-Size: {installed_kb}',
        f'Depends: {DEPENDS}',
        'Description: GuildSync voice-channel mute companion',
        ' Press-to-mute desktop companion for the GuildSync Discord bot. It signs in',
        ' with Discord and asks the existing GuildSync backend to mute lower-ranked',
        ' members in your current voice channel for a short, self-ending interval.',
    ]
    return '\n'.join(fields) + '\n'


def _copy_binary(root, destination):
    source = Path(root) / 'bin' / BINARY
    if not source.is_file():
        raise ValueError(f'the Linux payload is missing: {source}')
    shutil.copy2(source, destination)
    Path(destination).chmod(0o755)


def linux_package(root, version, architecture='amd64'):
    """Build a .deb that installs the binary, a desktop entry and an icon."""
    if not shutil.which('dpkg-deb'):
        raise ValueError('dpkg-deb is required to build the .deb package')
    root = Path(root)
    stage = Path(tempfile.mkdtemp())
    try:
        binary = stage / 'usr' / 'bin' / SLUG
        binary.parent.mkdir(parents=True)
        _copy_binary(root, binary)

        entry = stage / 'usr' / 'share' / 'applications' / f'{SLUG}.desktop'
        entry.parent.mkdir(parents=True)
        shutil.copy2(root / 'build' / 'linux' / f'{SLUG}.desktop', entry)

        icon = stage / 'usr' / 'share' / 'icons' / 'hicolor' / '256x256' / 'apps' / f'{SLUG}.png'
        icon.parent.mkdir(parents=True)
        shutil.copy2(root / 'build' / 'appicon.png', icon)

        control = stage / 'DEBIAN' / 'control'
        control.parent.mkdir(parents=True)
        installed_kb = (binary.stat().st_size + icon.stat().st_size) // 1024 + 1
        control.write_text(deb_control(version, installed_kb, architecture))

        archive = release_dir(root) / f'{BINARY}-{version}-Linux.deb'
        subprocess.run(['dpkg-deb', '--build', '--root-owner-group', str(stage), str(archive)], check=True)
        return archive
    finally:
        shutil.rmtree(stage, ignore_errors=True)


def macos_installer(root, version):
    """Build a .dmg holding the .app bundle and an Applications link."""
    if not shutil.which('hdiutil'):
        raise ValueError('a .dmg can only be built on macOS (hdiutil is missing)')
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
    stage = Path(tempfile.mkdtemp())
    try:
        shutil.copytree(bundle, stage / bundle.name, symlinks=True)
        (stage / 'Applications').symlink_to('/Applications')
        archive = release_dir(root) / f'{BINARY}-{version}-macOS.dmg'
        subprocess.run(
            ['hdiutil', 'create', '-volname', 'GuildSync Voice Mute', '-srcfolder', str(stage),
             '-ov', '-format', 'UDZO', str(archive)],
            check=True,
        )
        return archive
    finally:
        shutil.rmtree(stage, ignore_errors=True)


def find_iscc():
    override = os.environ.get('INNO_SETUP_ISCC')
    candidates = [override, shutil.which('ISCC.exe'), shutil.which('iscc'), *ISCC_CANDIDATES]
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
    parser.add_argument('--arch', default='amd64', help='Debian architecture for the .deb')
    parser.add_argument('--iscc', help='Path to Inno Setup ISCC.exe')
    arguments = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    try:
        version = metadata.validate(root, arguments.version)
        if arguments.platform == 'linux':
            archive = linux_package(root, version, arguments.arch)
        elif arguments.platform == 'macos':
            archive = macos_installer(root, version)
        else:
            archive = windows_installer(root, version, arguments.iscc)
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f'::error::installer packaging failed: {error}')
        return 1
    print(archive)
    return 0


if __name__ == '__main__':
    sys.exit(main())
