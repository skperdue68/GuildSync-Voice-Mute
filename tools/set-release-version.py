"""Stamp release metadata before building; no credentials or runtime files are touched."""
import json
import re
import sys
from pathlib import Path

# build/config.yml carries the product version under `info:`; the schema version
# at the top of the file is at column 0 and is deliberately left alone.
CONFIG_VERSION = re.compile(r'^(  version: )[^\n]*$', re.MULTILINE)


def _stamp_config(text, version):
    updated, count = CONFIG_VERSION.subn(lambda match: match.group(1) + '"' + version + '"', text)
    if count != 1:
        raise ValueError('build/config.yml must contain exactly one info.version entry')
    return updated


def _stamp_plist(text, version):
    for key in ('CFBundleVersion', 'CFBundleShortVersionString'):
        pattern = re.compile(r'(<key>' + key + r'</key>\s*<string>)[^<]*(</string>)')
        text, count = pattern.subn(lambda match: match.group(1) + version + match.group(2), text)
        if count == 0:
            raise ValueError('missing ' + key + ' in plist')
    return text


def _stamp_windows_metadata(data, version):
    fixed = data.setdefault('fixed', {})
    fixed['file_version'] = version
    fixed['product_version'] = version
    for locale in data.get('info', {}).values():
        locale['FileVersion'] = version
        locale['ProductVersion'] = version
    return data


def stamp(root, tag):
    version = tag.removeprefix('v')
    if not re.fullmatch(r'\d+\.\d+\.\d+', version):
        raise ValueError('Release tag must be vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH')

    config_path = root / 'build' / 'config.yml'
    windows_path = root / 'build' / 'windows' / 'info.json'
    plist_paths = [root / 'build' / 'darwin' / 'Info.plist', root / 'build' / 'darwin' / 'Info.dev.plist']
    package_paths = [root / 'frontend' / 'package.json', root / 'frontend' / 'package-lock.json']

    documents = [(path, json.loads(path.read_text())) for path in [windows_path] + package_paths]
    config_text = _stamp_config(config_path.read_text(), version)
    plists = [(path, _stamp_plist(path.read_text(), version)) for path in plist_paths]

    config_path.write_text(config_text)
    for path, text in plists:
        path.write_text(text)
    for path, data in documents:
        if path.name == 'info.json':
            _stamp_windows_metadata(data, version)
        else:
            data['version'] = version
            if path.name == 'package-lock.json':
                data['packages']['']['version'] = version
        path.write_text(json.dumps(data, indent=2) + '\n')
    return version


if __name__ == '__main__':
    print('Release version: ' + stamp(Path(__file__).resolve().parents[1], sys.argv[1]))
