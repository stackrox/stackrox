#!/usr/bin/env python3

import argparse
import sys
from datetime import datetime, timezone

import yaml


def replace_suffix(str, suffix, replacement):
    splits = str.rsplit(suffix, maxsplit=1)
    if len(splits) == 1:
        raise RuntimeError(str + " does not contain " + suffix)
    return splits[0] + replacement


def patch_csv(csv_doc, version):
    # TODO(ROX-7165): configure update path
    csv_doc['metadata']['annotations']['createdAt'] = datetime.now(timezone.utc).isoformat()

    csv_doc['metadata']['annotations']['containerImage'] = \
        replace_suffix(csv_doc['metadata']['annotations']['containerImage'], ':0.0.1', ':' + version)

    csv_doc['metadata']['name'] = \
        replace_suffix(csv_doc['metadata']['name'], '.v0.0.1', '.v' + version)

    for deployment in csv_doc['spec']['install']['spec']['deployments']:
        for container in deployment['spec']['template']['spec']['containers']:
            if "-operator:" not in container['image']:
                continue
            container['image'] = replace_suffix(container['image'], ':0.0.1', ':' + version)

    csv_doc['spec']['version'] = version


def parse_args():
    parser = argparse.ArgumentParser(description='Patch StackRox Operator ClusterServiceVersion file')
    parser.add_argument("--use-version", required=True, metavar='version',
                        help='Which SemVer version of the operator to set in the patched CSV, e.g. 3.62.0')
    return parser.parse_args()


def main():
    args = parse_args()
    doc = yaml.safe_load(sys.stdin)
    patch_csv(doc, args.use_version)
    print(yaml.safe_dump(doc))


if __name__ == '__main__':
    main()
