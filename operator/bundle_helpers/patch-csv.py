#!/usr/bin/env python3

import argparse
import logging
import os
import pathlib
import subprocess
import sys
import yaml
from collections import namedtuple
from datetime import datetime, timezone

from rewrite import rewrite, string_replacer


class XyzVersion(namedtuple("Version", ["x", "y", "z"])):
    @staticmethod
    def parse_from(version_str):
        x, y, z = (int(c) for c in version_str.split('-', maxsplit=1)[0].split('.'))
        return XyzVersion(x, y, z)

    def __str__(self):
        return f"{self.x}.{self.y}.{self.z}"


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


def patch_csv(csv_doc, version, operator_image, first_version, no_related_images, extra_supported_arches,
              rbac_proxy_replacement):
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

    previous_y_stream = get_previous_y_stream(version)

    # An olm.skipRange doesn't hurt if it references non-existing versions.
    csv_doc["metadata"]["annotations"]["olm.skipRange"] = f'>= {previous_y_stream} < {version}'

    # multi-arch
    if "labels" not in csv_doc["metadata"]:
        csv_doc["metadata"]["labels"] = {}
    for arch in extra_supported_arches:
        csv_doc["metadata"]["labels"][f"operatorframework.io/arch.{arch}"] = "supported"

    skips = parse_skips(csv_doc["spec"], raw_name)
    replaced_xyz = calculate_replaced_version(
        version=version, first_version=first_version, previous_y_stream=previous_y_stream, skips=skips)
    if replaced_xyz is not None:
        csv_doc["spec"]["replaces"] = f"{raw_name}.v{replaced_xyz}"

    # OSBS fills relatedImages therefore we must not provide that ourselves.
    # Ref https://osbs.readthedocs.io/en/latest/users.html?highlight=relatedImages#creating-the-relatedimages-section
    del csv_doc['spec']['relatedImages']


def parse_skips(spec, raw_name):
    raw_skips = spec.get("skips", [])
    return set([XyzVersion.parse_from(must_strip_prefix(item, f"{raw_name}.v")) for item in raw_skips])


def must_strip_prefix(str, prefix):
    if not str.startswith(prefix):
        raise RuntimeError(f"{str} does not begin with {prefix}")
    return str[len(prefix):]


def calculate_replaced_version(version, first_version, previous_y_stream, skips):
    current_xyz = XyzVersion.parse_from(version)
    first_xyz = XyzVersion.parse_from(first_version)
    previous_xyz = XyzVersion.parse_from(previous_y_stream)

    if current_xyz <= first_xyz:
        return None

    initial_replace = previous_xyz if current_xyz.z == 0 else \
        XyzVersion(current_xyz.x, current_xyz.y, current_xyz.z - 1)

    current_replace = initial_replace

    while current_replace in skips:
        logging.info(f"Looks like {current_replace} replace version is in skips list, trying next patch.")
        current_replace = XyzVersion(current_replace.x, current_replace.y, current_replace.z + 1)

    if current_replace >= current_xyz:
        current_replace = initial_replace
        logging.warning(
            f"Cannot identify safe patch version among skips {skips} that would be less than current {current_xyz}. Falling back to original {current_replace}.")

    return current_replace


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
    logging.basicConfig(stream=sys.stderr, level=logging.INFO,
                        format=f"%(asctime)s {pathlib.Path(__file__).name}: %(message)s")
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
