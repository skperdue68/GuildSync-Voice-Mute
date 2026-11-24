"""Verify the installer a release build just produced, on its own platform.

A release is only published once this has run in the build job, so a tag cannot
become a public version whose installer was merely built. The checks are the ones
the pull-request smoke job uses (run the setup silently, mount the DMG, extract
the AppImage), applied here to the real installer in release/.
"""
import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import smoke_installers

# The installer suffixes the updater looks for, so this checks the file the
# in-app update will actually download rather than something beside it.
INSTALLERS = {
    'windows': '-Windows-Setup.exe',
    'macos': '-macOS.dmg',
    'linux': '.AppImage',
}


def find_installer(directory, platform):
    """The one installer this build produced for the platform."""
    suffix = INSTALLERS[platform]
    matches = sorted(path for path in Path(directory).glob('*' + suffix) if path.is_file())
    if len(matches) != 1:
        found = ', '.join(path.name for path in matches) or 'none'
        raise ValueError(f'expected one *{suffix} installer in {directory}, found {found}')
    return matches[0]


def verify(directory, platform):
    archive = find_installer(directory, platform)
    if platform == 'windows':
        smoke_installers.verify_windows(archive)
    elif platform == 'macos':
        smoke_installers.verify_macos(archive)
    else:
        # The download is opened as a program, so the published file has to be
        # executable before its contents can be checked.
        archive.chmod(0o755)
        smoke_installers.verify_linux(archive)
    return archive


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('platform', choices=tuple(INSTALLERS))
    parser.add_argument('--release-dir', default='release', help='Directory the build packaged into')
    arguments = parser.parse_args()
    try:
        archive = verify(arguments.release_dir, arguments.platform)
    except (OSError, ValueError) as error:
        print(f'::error::installer verification failed: {error}')
        return 1
    print(f'verified {archive}')
    return 0


if __name__ == '__main__':
    sys.exit(main())
