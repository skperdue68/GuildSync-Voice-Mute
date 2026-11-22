"""Shared product-metadata validation for the PR gate and release packaging.

The version appears in build/config.yml (read by the Go binary), the Windows
resource metadata, both macOS plists and the frontend package files. A release
tag stamps all of them; this module proves a pull request has not let them drift
apart, and package_release.py refuses to build an archive when they have.
"""
import json
import re
from pathlib import Path

PRODUCT_NAME = 'GuildSync Voice Mute'
PRODUCT_IDENTIFIER = 'me.perdues.guildsync.voicemute'
RELEASE_VERSION = re.compile(r'\d+\.\d+\.\d+$')
# build/config.yml carries the product version under `info:` with a trailing
# comment; the schema version at the top of the file has no indentation.
CONFIG_VERSION = re.compile(r'^  version:\s*"([^"]+)"', re.MULTILINE)


def read_config_version(root):
    matches = CONFIG_VERSION.findall((Path(root) / 'build' / 'config.yml').read_text())
    if len(matches) != 1:
        raise ValueError('build/config.yml must contain exactly one info.version entry')
    return matches[0]


def _plist_value(text, key):
    match = re.search(r'<key>' + re.escape(key) + r'</key>\s*<string>([^<]*)</string>', text)
    if not match:
        raise ValueError('missing ' + key + ' in macOS plist')
    return match.group(1)


def validate(root, expected=None):
    """Return the product version, raising ValueError when anything disagrees."""
    root = Path(root)
    version = read_config_version(root)
    if not RELEASE_VERSION.fullmatch(version):
        raise ValueError(f'build/config.yml version {version!r} is not MAJOR.MINOR.PATCH')

    package = json.loads((root / 'frontend' / 'package.json').read_text())
    if package.get('version') != version:
        raise ValueError('frontend/package.json version does not match build/config.yml')

    lock = json.loads((root / 'frontend' / 'package-lock.json').read_text())
    if lock.get('version') != version or lock.get('packages', {}).get('', {}).get('version') != version:
        raise ValueError('frontend/package-lock.json version does not match build/config.yml')

    windows = json.loads((root / 'build' / 'windows' / 'info.json').read_text())
    fixed = windows.get('fixed', {})
    if fixed.get('product_version') != version or fixed.get('file_version') != version:
        raise ValueError('build/windows/info.json was not stamped with the product version')
    locales = windows.get('info', {})
    if not locales:
        raise ValueError('build/windows/info.json has no locale metadata')
    for locale, values in locales.items():
        if values.get('FileVersion') != version or values.get('ProductVersion') != version:
            raise ValueError(f'build/windows/info.json locale {locale} was not stamped')
        if values.get('ProductName') != PRODUCT_NAME:
            raise ValueError(f'build/windows/info.json locale {locale} has the wrong product name')

    for name in ('Info.plist', 'Info.dev.plist'):
        text = (root / 'build' / 'darwin' / name).read_text()
        if _plist_value(text, 'CFBundleShortVersionString') != version:
            raise ValueError(f'build/darwin/{name} CFBundleShortVersionString is stale')
        if _plist_value(text, 'CFBundleVersion') != version:
            raise ValueError(f'build/darwin/{name} CFBundleVersion is stale')
        if _plist_value(text, 'CFBundleIdentifier') != PRODUCT_IDENTIFIER:
            raise ValueError(f'build/darwin/{name} has the wrong bundle identifier')
        if _plist_value(text, 'CFBundleName') != PRODUCT_NAME:
            raise ValueError(f'build/darwin/{name} has the wrong bundle name')

    if expected is not None:
        wanted = str(expected).removeprefix('v')
        if wanted != version:
            raise ValueError(f'release tag must match the product version ({version})')
    return version
