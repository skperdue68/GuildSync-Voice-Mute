"""Point apt at the standard Ubuntu mirrors before it contacts the network.

GitHub-hosted Ubuntu runners rank azure.archive.ubuntu.com ahead of the standard
mirrors in /etc/apt/apt-mirrors.txt. That mirror sometimes serves package
archives at tens of kilobytes per second while the indexes still download fast,
and apt only gives up when no data arrives at all, so a crawling mirror is never
abandoned: the same install that takes 20 seconds on a healthy runner has taken
22-23 minutes here. See https://github.com/actions/runner-images/issues/14823.

Everything below is a local file edit, so this runs before apt's first network
call and cannot itself be blocked by the mirror it repairs. Run it as root:

    sudo python3 tools/apt_mirrors.py
"""
import sys
from pathlib import Path

DEFAULT_ROOT = Path('/etc/apt')
AZURE = 'azure.archive.ubuntu.com'
ARCHIVE = 'archive.ubuntu.com'
MIRROR_LIST = 'apt-mirrors.txt'
# apt's mirror+file: source needs at least one entry, so an image that listed
# only the Azure mirror falls back to the standard pair.
FALLBACK_MIRRORS = ARCHIVE + '/ubuntu/\tpriority:1\n' + 'security.ubuntu.com/ubuntu/\tpriority:2\n'


def source_files(root):
    """Files that can name a mirror, without descending into the binary keyrings."""
    candidates = [root / 'sources.list']
    candidates += sorted((root / 'sources.list.d').glob('*.list'))
    candidates += sorted((root / 'sources.list.d').glob('*.sources'))
    return [path for path in candidates if path.is_file()]


def rewrite(root=DEFAULT_ROOT):
    """Drop the Azure mirror under root and return one message per file changed."""
    root = Path(root)
    changes = []

    mirrors = root / MIRROR_LIST
    if mirrors.is_file():
        lines = mirrors.read_text().splitlines(keepends=True)
        kept = [line for line in lines if AZURE not in line]
        if len(kept) != len(lines):
            mirrors.write_text(''.join(kept) or FALLBACK_MIRRORS)
            changes.append(f'{mirrors}: dropped {AZURE}')

    for path in source_files(root):
        text = path.read_text()
        if AZURE in text:
            path.write_text(text.replace(AZURE, ARCHIVE))
            changes.append(f'{path}: {AZURE} -> {ARCHIVE}')

    return changes


def main(root=DEFAULT_ROOT):
    changes = rewrite(root)
    for change in changes:
        print(change)
    if not changes:
        print(f'apt already avoids {AZURE} under {root}')
    return 0


if __name__ == '__main__':
    sys.exit(main(*sys.argv[1:2]))
