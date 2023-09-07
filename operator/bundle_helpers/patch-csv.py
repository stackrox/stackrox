#!/usr/bin/env python3

import argparse
import logging
import os
import pathlib
import subprocess
import sys
import yaml
from datetime import datetime, timezone

from rewrite import rewrite, string_replacer


def rbac_proxy_replace(updated_img):
    def update_rbac_proxy_img(img):
        """
        Updates the reference to the kube-rbac-proxy image to match the OpenShift one.
        """
        if not isinstance(img, str) or not img.startswith('gcr.io/kubebuilder/kube-rbac-proxy:'):
            return None
        return updated_img

    return update_rbac_proxy_img


def related_image_passthrough(val):
    """
    Searches for environment variable definitions of the form RELATED_IMAGE_* and replaces them
    from the current environment. It is an error if one of the environment variables does not
    exist in the environment.
    """
    if not isinstance(val, dict):
        return None
    name = val.get("name")
    if not isinstance(name, str):
        return None
    if name.startswith("RELATED_IMAGE_"):
        val["value"] = os.environ[name]


def must_replace_suffix(str, suffix, replacement):
    """
    Replaces the given suffix in the string. If the string does not have the suffix, a runtime
    error will be raised.
    """
    splits = str.rsplit(suffix, maxsplit=1)
    if len(splits) != 2 or splits[1]:
        raise RuntimeError(str + " does not contain " + suffix)
    return splits[0] + replacement


def patch_csv(csv_doc, version, operator_image, first_version, no_related_images, extra_supported_arches, rbac_proxy_replacement):
    csv_doc['metadata']['annotations']['createdAt'] = datetime.now(timezone.utc).isoformat()

    placeholder_image = csv_doc['metadata']['annotations']['containerImage']
    rewrite(csv_doc, string_replacer(placeholder_image, operator_image))

    raw_name = must_replace_suffix(csv_doc['metadata']['name'], '.v0.0.1', '')
    csv_doc['metadata']['name'] = f'{raw_name}.v{version}'

    csv_doc['spec']['version'] = version

    if not no_related_images:
        rewrite(csv_doc, related_image_passthrough)

    if rbac_proxy_replacement:
        rewrite(csv_doc, rbac_proxy_replace(rbac_proxy_replacement))

    x, y, z = (int(c) for c in version.split('-', maxsplit=1)[0].split('.'))
    first_x, first_y, first_z = (int(c) for c in first_version.split('-', maxsplit=1)[0].split('.'))
    previous_y_stream = get_previous_y_stream(version)

    # An olm.skipRange doesn't hurt if it references non-existing versions.
    csv_doc["metadata"]["annotations"]["olm.skipRange"] = f'>= {previous_y_stream} < {version}'

    # multi-arch
    if "labels" not in csv_doc["metadata"]:
        csv_doc["metadata"]["labels"] = {}
    for arch in extra_supported_arches:
        csv_doc["metadata"]["labels"][f"operatorframework.io/arch.{arch}"] = "supported"

    if (x, y, z) > (first_x, first_y, first_z):
        if z == 0:
            csv_doc["spec"]["replaces"] = f'{raw_name}.v{previous_y_stream}'
        else:
            csv_doc["spec"]["replaces"] = f'{raw_name}.v{x}.{y}.{z - 1}'

    # OSBS fills relatedImages therefore we must not provide that ourselves.
    # Ref https://osbs.readthedocs.io/en/latest/users.html?highlight=relatedImages#creating-the-relatedimages-section
    del csv_doc['spec']['relatedImages']


def get_previous_y_stream(version):
    this_script_dir = pathlib.Path(__file__).parent
    executable = this_script_dir / "../../scripts/get-previous-y-stream.sh"
    # subprocess.run()'s  capture_output=True argument first appeared in Python 3.7 which is not available universally
    # (e.g. in our upstream builder image), therefore we capture stdout with a bit dated check_output() call.
    return subprocess.check_output([executable, version], encoding='utf-8').strip()


def parse_args():
    parser = argparse.ArgumentParser(description='Patch StackRox Operator ClusterServiceVersion file')
    parser.add_argument("--use-version", required=True, metavar='version',
                        help='Which SemVer version of the operator to set in the patched CSV, e.g. 3.62.0')
    parser.add_argument("--first-version", required=True, metavar='version',
                        help='The first version of the operator that was published')
    parser.add_argument("--operator-image", required=True, metavar='image',
                        help='Which operator image to use in the patched CSV')
    parser.add_argument("--no-related-images", action='store_true',
                        help='Disable passthrough of related images')
    parser.add_argument("--replace-rbac-proxy", required=False, metavar='replacement-image:tag',
                        help='Replacement directives for the RBAC proxy image')
    parser.add_argument("--add-supported-arch", action='append', required=False,
                        help='Enable specified operator architecture via CSV labels (may be passed multiple times)',
                        default=[])
    return parser.parse_args()


def main():
    logging.basicConfig(stream=sys.stderr, level=logging.INFO)
    args = parse_args()
    doc = yaml.safe_load(sys.stdin)
    patch_csv(doc,
              operator_image=args.operator_image,
              version=args.use_version,
              first_version=args.first_version,
              no_related_images=args.no_related_images,
              extra_supported_arches=args.add_supported_arch,
              rbac_proxy_replacement=args.replace_rbac_proxy)
    print(yaml.safe_dump(doc))


if __name__ == '__main__':
    main()
