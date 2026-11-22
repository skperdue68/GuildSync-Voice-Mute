"""Zip the built application for a release, validating metadata first.

Releases are zips of bin/ plus the user documentation; nothing is installed and
nothing outside the repository is touched.
"""
import argparse
import sys
import zipfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import metadata


def package(root, platform, version=None):
    root = Path(root)
    version = metadata.validate(root, version)
    bin_dir = root / 'bin'
    files = [path for path in bin_dir.rglob('*') if path.is_file()] if bin_dir.exists() else []
    if not files:
        raise ValueError('no executable was built in bin/')
    output = root / 'release'
    output.mkdir(exist_ok=True)
    archive = output / f'GuildSync-Voice-Mute-{version}-{platform}.zip'
    with zipfile.ZipFile(archive, 'w', zipfile.ZIP_DEFLATED) as bundle:
        for path in files:
            bundle.write(path, path.relative_to(bin_dir))
        bundle.write(root / 'README.md', 'README.md')
        bundle.write(root / 'docs' / 'platform-support.md', 'docs/platform-support.md')
    return archive


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('platform', help='Archive platform label, e.g. Linux')
    parser.add_argument('--version', help='Release tag the metadata must match')
    arguments = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    try:
        archive = package(root, arguments.platform, arguments.version)
    except (OSError, ValueError) as error:
        print(f'::error::release packaging failed: {error}')
        return 1
    print(archive)
    return 0


if __name__ == '__main__':
    sys.exit(main())
