"""Fail a pull request whose product metadata no longer agrees.

Prints the product version on success and a GitHub error annotation on failure.
"""
import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import metadata


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--expected', help='Release tag the version must match')
    arguments = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    try:
        version = metadata.validate(root, arguments.expected)
    except (OSError, ValueError) as error:
        print(f'::error::product metadata is inconsistent: {error}')
        return 1
    print(version)
    return 0


if __name__ == '__main__':
    sys.exit(main())
