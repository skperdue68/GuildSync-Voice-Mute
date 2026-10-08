"""Stamp release metadata before building; no credentials or runtime files are touched."""
import json
import re
import sys
from pathlib import Path

def stamp(root, tag):
    version = tag.removeprefix('v')
    if not re.fullmatch(r'\d+\.\d+\.\d+', version):
        raise ValueError('Release tag must be vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH')
    paths = ['wails.json', 'frontend/package.json', 'frontend/package-lock.json']
    documents = [(root / path, json.loads((root / path).read_text())) for path in paths]
    for path, data in documents:
        if path.name == 'wails.json':
            data['info']['productVersion'] = version
        else:
            data['version'] = version
            if path.name == 'package-lock.json':
                data['packages']['']['version'] = version
        path.write_text(json.dumps(data, indent=2) + '\n')
    return version

if __name__ == '__main__':
    print('Release version: ' + stamp(Path(__file__).resolve().parents[1], sys.argv[1]))
