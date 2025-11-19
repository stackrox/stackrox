#!/usr/bin/env python3

import argparse
import logging
import os
import pathlib
import subprocess
import sys
import textwrap
from collections import namedtuple
from datetime import datetime, timezone

import yaml

from rewrite import rewrite, string_replacer


class XyzVersion(namedtuple("Version", ["x", "y", "z"])):
    @staticmethod
    def parse_from(version_str):
        x, y, z = (int(c) for c in version_str.split('-', maxsplit=1)[0].split('.'))
        return XyzVersion(x, y, z)

    def __str__(self):
        return f"{self.x}.{self.y}.{self.z}"


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


def patch_csv(csv_doc, version, operator_image, first_version, related_images_mode, extra_supported_arches,
              unreleased=None):
    csv_doc['metadata']['annotations']['createdAt'] = datetime.now(timezone.utc).isoformat()

    placeholder_image = csv_doc['metadata']['annotations']['containerImage']
    rewrite(csv_doc, string_replacer(placeholder_image, operator_image))

    raw_name = must_replace_suffix(csv_doc['metadata']['name'], '.v0.0.1', '')
    csv_doc['metadata']['name'] = f'{raw_name}.v{version}'

    csv_doc['spec']['version'] = version

    if related_images_mode != "omit":
        rewrite(csv_doc, related_image_passthrough)

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
        version=version, first_version=first_version, previous_y_stream=previous_y_stream, skips=skips,
        unreleased=unreleased)
    if replaced_xyz is not None:
        csv_doc["spec"]["replaces"] = f"{raw_name}.v{replaced_xyz}"

    if related_images_mode == "konflux":
        csv_doc['spec']['relatedImages'] = construct_related_images(operator_image)
    elif 'relatedImages' in csv_doc['spec']:
        # OSBS fills relatedImages therefore we must not provide that ourselves.
        # Ref https://osbs.readthedocs.io/en/latest/users.html?highlight=relatedImages#creating-the-relatedimages-section
        del csv_doc['spec']['relatedImages']

    # Add SecurityPolicy CRD to ACS operator CSV
    policy_crd = {
        "name": "securitypolicies.config.stackrox.io",
        "version": "v1alpha1",
        "kind": "SecurityPolicy",
        "displayName": "Security Policy",
        "description": "SecurityPolicy is the schema for the policies API.",
        "resources": [{
            "kind": "Deployment",
            "name": "",
            "version": "v1",
        }],
    }

    csv_doc["spec"]["customresourcedefinitions"]["owned"].append(policy_crd)

def construct_related_images(manager_image):
    related_images = []
    for name, image in os.environ.items():
        if name.startswith("RELATED_IMAGE_"):
            name = name.removeprefix("RELATED_IMAGE_")
            name = name.lower()
            related_images.append({'name': name, 'image': image})
    # Also inject the "manager" related image, which should be listed in `relatedImages` for the purpose of
    # air-gapped installation, but has no reason to appear in operator manager's environment.
    related_images.append({'name': 'manager', 'image': manager_image})
    return related_images


def parse_skips(spec, raw_name):
    raw_skips = spec.get("skips", [])
    return set([XyzVersion.parse_from(must_strip_prefix(item, f"{raw_name}.v")) for item in raw_skips])


def must_strip_prefix(str, prefix):
    if not str.startswith(prefix):
        raise RuntimeError(f"{str} does not begin with {prefix}")
    return str[len(prefix):]


def calculate_replaced_version(version, first_version, previous_y_stream, skips, unreleased=None):
    current_xyz = XyzVersion.parse_from(version)
    first_xyz = XyzVersion.parse_from(first_version)
    previous_xyz = XyzVersion.parse_from(previous_y_stream)

    if current_xyz <= first_xyz:
        return None

    # If this is a new minor release, it will replace the previous minor release (e.g. 4.2.0 replaces 4.1.0).
    # If this is a new patch, it replaces previous patch (e.g. 4.2.2 replaces 4.2.1, or 4.2.1 replaces 4.2.0).
    initial_replace = previous_xyz if current_xyz.z == 0 else \
        XyzVersion(current_xyz.x, current_xyz.y, current_xyz.z - 1)

    # If this version is not yet released, try previous one.
    # E.g. 4.5 branch was cut and the 4.6.x tag created, but the 4.5 release process is still in progress.
    if unreleased and str(initial_replace) == str(unreleased):
        initial_replace = XyzVersion.parse_from(get_previous_y_stream(str(initial_replace)))

    # Next, in the presence of version skips, i.e. versions that are marked as broken with `skips` attribute, we need to
    # handle a situation when the replaced version is also skipped, because the upgrade may fail.

    current_replace = initial_replace

    # First, we loop over all skips and find a patch number that's not skipped. This assumes there always exists a
    # released patch for any version that's skipped. E.g. we release 4.2.0, and 4.1.0 is broken and listed in `skips`,
    # and so we cannot make 4.2.0 replace 4.1.0. We'll take 4.2.0 replace 4.1.1.
    # The assumption should hold true because if some version is determined broken and is supported, we should create a
    # patch release to fix it.
    while current_replace in skips:
        logging.info(f"Looks like {current_replace} replace version is in skips list, trying next patch.")
        current_replace = XyzVersion(current_replace.x, current_replace.y, current_replace.z + 1)

    # The obvious exception from the above is when we release the immediate patch to the broken version.
    # E.g. 4.1.0 is broken and listed in `skips` and we release 4.1.1. In this case 4.1.1 will still replace 4.1.0. The
    # operator upgrade is still possible because 4.1.1 will additionally have skipRange >=4.0.0 and <4.1.1 thus allowing
    # versions in that range to be upgraded to 4.1.1.
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


# This class configures ArgumentParser help to print default values and preserve linebreaks in argument help.
class HelpFormatter(argparse.ArgumentDefaultsHelpFormatter, argparse.RawTextHelpFormatter):
    pass


def parse_args():
    parser = argparse.ArgumentParser(description='Patch StackRox Operator ClusterServiceVersion file',
                                     formatter_class=HelpFormatter)
    parser.add_argument("--use-version", required=True, metavar='version',
                        help='Which SemVer version of the operator to set in the patched CSV, e.g. 3.62.0')
    parser.add_argument("--first-version", required=True, metavar='version',
                        help='The first version of the operator that was published')
    parser.add_argument("--operator-image", required=True, metavar='image',
                        help='Which operator image to use in the patched CSV')
    parser.add_argument("--related-images-mode", choices=["downstream", "omit", "konflux"], default="downstream",
                        help=textwrap.dedent("""
                        Set mode of operation for handling related image attributes in the output CSV.
                        Supported modes:
                            downstream: In this mode the current RELATED_IMAGE_* environment variables are injected into
                                the output CSV and spec.relatedImages is not added.
                            omit: In this mode no RELATED_IMAGE_* environment variables are injected into the output CSV
                                and spec.relatedImages is not added.
                            konflux: In this mode the current RELATED_IMAGE_* environment variables are injected into the
                                output CSV and spec.relatedImages is populated based on them.
                        """).lstrip())
    parser.add_argument("--add-supported-arch", action='append', required=False,
                        help='Enable specified operator architecture via CSV labels (may be passed multiple times)',
                        default=["amd64", "arm64", "ppc64le", "s390x"])
    parser.add_argument("--echo-replaced-version-only", action='store_true',
                        help='Do not modify any files, just compute and echo the replaced operator version.')
    parser.add_argument("--unreleased", help="Not yet released version of operator, if any.")
    return parser.parse_args()


def main():
    logging.basicConfig(stream=sys.stderr, level=logging.INFO,
                        format=f"%(asctime)s {pathlib.Path(__file__).name}: %(message)s")
    args = parse_args()
    doc = yaml.safe_load(sys.stdin)
    if args.echo_replaced_version_only:
        raw_name = must_replace_suffix(doc['metadata']['name'], '.v0.0.1', '')
        skips = parse_skips(doc["spec"], raw_name)
        replaced_xyz = calculate_replaced_version(
            version=args.use_version, first_version=args.first_version,
            previous_y_stream=get_previous_y_stream(args.use_version), skips=skips)
        print(replaced_xyz)
        return
    patch_csv(doc,
              operator_image=args.operator_image,
              version=args.use_version,
              first_version=args.first_version,
              unreleased=args.unreleased,
              related_images_mode=args.related_images_mode,
              extra_supported_arches=args.add_supported_arch)
    print(yaml.safe_dump(doc))


if __name__ == '__main__':
    main()
