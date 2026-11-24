"""Point apt at Canonical's archive mirror before it contacts the network.

GitHub's Ubuntu runner images rank a regional copy of the Ubuntu archive ahead
of Canonical's own: ``azure.archive.ubuntu.com`` on some images and
``us-east-2.ec2.archive.ubuntu.com`` on the ubuntu-24.04 image a run used on
2026-11-24. Those regional hosts can serve at a fraction of normal speed -- the
same index file measured 178 kB/s from the EC2 mirror and 19.7 MB/s from
archive.ubuntu.com, and a runner's apt log showed 403 kB/s -- and apt only gives
up when no data arrives at all, so a crawling mirror is never abandoned. The
GTK4/WebKitGTK install took 14-23 minutes that way against 20 seconds on a
runner whose mirror was healthy.

Everything below is a local file edit, so this runs before apt's first network
call and cannot itself be blocked by the mirror it repairs. Run it as root:

    sudo python3 tools/apt_mirrors.py
"""
import re
import sys
from pathlib import Path

DEFAULT_ROOT = Path('/etc/apt')
ARCHIVE = 'archive.ubuntu.com'
MIRROR_LIST = 'apt-mirrors.txt'
# A host that carries the Ubuntu archive's name but is not Canonical's own, such
# as azure.archive.ubuntu.com or us-east-2.ec2.archive.ubuntu.com. The leading
# labels are required, so archive.ubuntu.com itself never matches.
REGIONAL_ARCHIVE = re.compile(r'(?:[A-Za-z0-9-]+\.)+archive\.ubuntu\.com')


def source_files(root):
    """Files that can name a mirror, without descending into the binary keyrings."""
    candidates = [root / 'sources.list']
    candidates += sorted((root / 'sources.list.d').glob('*.list'))
    candidates += sorted((root / 'sources.list.d').glob('*.sources'))
    return [path for path in candidates if path.is_file()]


def _canonical(text):
    return REGIONAL_ARCHIVE.sub(ARCHIVE, text)


def _dedupe_mirrors(text):
    """Keep the first line for each mirror, so repointing cannot repeat one."""
    kept = []
    seen = set()
    for line in text.splitlines(keepends=True):
        fields = line.split()
        if fields:
            if fields[0] in seen:
                continue
            seen.add(fields[0])
        kept.append(line)
    return ''.join(kept)


def rewrite(root=DEFAULT_ROOT):
    """Repoint regional mirrors under root at Canonical, one message per change."""
    root = Path(root)
    changes = []

    mirrors = root / MIRROR_LIST
    if mirrors.is_file():
        text = mirrors.read_text()
        updated = _dedupe_mirrors(_canonical(text))
        if updated != text:
            mirrors.write_text(updated)
            changes.append(f'{mirrors}: repointed the regional archive mirror at {ARCHIVE}')

    for path in source_files(root):
        text = path.read_text()
        updated = _canonical(text)
        if updated != text:
            path.write_text(updated)
            changes.append(f'{path}: repointed the regional archive mirror at {ARCHIVE}')

    return changes


def main(root=DEFAULT_ROOT):
    changes = rewrite(root)
    for change in changes:
        print(change)
    if not changes:
        print(f'apt already uses {ARCHIVE} under {root}')
    return 0


if __name__ == '__main__':
    sys.exit(main(*sys.argv[1:2]))
