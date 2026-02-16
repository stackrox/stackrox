#!/usr/bin/env python3

"""
Returns the central-sensor version tuples to be used for compatibility testing.
"""
import logging
import subprocess
import sys
from collections import namedtuple
from pathlib import Path

from get_latest_helm_chart_versions import (
    get_supported_helm_chart_versions,
    get_latest_helm_chart_version_for_specific_release,
)


# We run compatibility tests against supported older versions of Stackrox.
# get_compatibility_test_tuples.py provides the function get_compatibility_test_tuples() which is called in our
# compatibility tests that returns all tuples of central and sensor versions for which:
#    1. Central is latest and Sensor is a supported older version OR
#    2. Sensor is latest and Central is a supported older version OR
#    3. Is a support exception
# These are returned only if Helm charts for both versions could be found.
# After running get_compatibility_test_tuples.py I received the following output:
# INFO:root:Listing supported versions tuples:
# INFO:root:Tuple 1: {Central v4.11.x-94-g75a2cb6b34 - Sensor v40009.5.3}
# INFO:root:Tuple 2: {Central v4.11.x-94-g75a2cb6b34 - Sensor v40008.1.0}
# INFO:root:Tuple 3: {Central v40009.5.3 - Sensor v4.11.x-94-g75a2cb6b34}
# INFO:root:Tuple 4: {Central v40008.1.0 - Sensor v4.11.x-94-g75a2cb6b34}
# If no supported versions with available Helm charts are found, an empty list is returned.
def main():
    logging.basicConfig(stream=sys.stderr, level=logging.DEBUG)
    test_tuples = get_compatibility_test_tuples()
    logging.info(
        "Listing supported versions tuples:"
    )
    i = 0
    for test_tuple in test_tuples:
        i += 1
        logging.info(
            "Tuple %s: {Central v%s - Sensor v%s}", str(i), test_tuple.central_version, test_tuple.sensor_version
        )


# Returns True if the helm_version is newer than the current_version
def is_newer_version(current_version: str, helm_version: str):
    helm_version_split = helm_version.split(sep='.')
    current_version_split = current_version.split(sep='.')

    # Parse helm version format using numeric encoding:
    # - New format: X00MM where X=major digit, 00=padding, MM=minor (zero-padded)
    #   Example: 40009 → major=4, minor=09 (version 4.9)
    #   Example: 40011 → major=4, minor=11 (version 4.11)
    #   Formula: major = n // 10000, minor = n % 100
    # - Old format: X00 where X=major digit, 00=padding (version 4.0.x)
    #   Formula: major = n // 100
    helm_major_num = int(helm_version_split[0])
    if helm_major_num >= 10000:
        # New format: extract major (first digit) and minor (last 2 digits)
        helm_major = helm_major_num // 10000
        helm_minor = helm_major_num % 100
        helm_version_split = [str(helm_major), str(helm_minor)] + helm_version_split[1:]
    else:
        # Old format: extract major by removing padding
        helm_version_split[0] = str(helm_major_num // 100)

    # Remove commit hash from the current version
    current_version_split = current_version_split[:-1]
    # If we are in a release branch, we will have patch version with '-rc'
    if len(current_version_split) > 2:
        # Remove '-rc' if present
        current_version_split[2] = str(current_version_split[2]).rstrip("-rc")

    for (current, helm) in zip(current_version_split, helm_version_split):
        if int(current) > int(helm):
            break
        if int(current) < int(helm):
            return True

    return False


def get_compatibility_test_tuples():
    Release = namedtuple("Release", ["major", "minor"])

    # start logging
    logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

    central_chart_versions, sensor_chart_versions = get_supported_helm_chart_versions()

    makefile_path = Path(__file__).parent.parent
    latest_tag = subprocess.check_output(
        ["make", "tag", "-C", makefile_path, "--quiet", "--no-print-director"],
        shell=False,
        encoding="utf-8",
    ).strip()

    # Remove the versions that are newer than the version of the current branch.
    # This will make sure we do not test with an old test suite newer versions.
    # It is important to not test newer versions with old tests suites because
    # old test suites might depend on endpoints that no longer exist in newer
    # versions.
    # There is no risk in excluding newer versions as the compatibility tests in
    # their respective branches will test against older versions.
    central_chart_versions = [i for i in central_chart_versions
                              if not
                              is_newer_version(current_version=latest_tag,
                                               helm_version=i)]
    sensor_chart_versions = [i for i in sensor_chart_versions
                             if not
                             is_newer_version(current_version=latest_tag,
                                              helm_version=i)]

    if len(central_chart_versions) == 0:
        logging.info("Found no older central chart versions to test against according to the product lifecycles API.")
    if len(sensor_chart_versions) == 0:
        logging.info("Found no older sensor chart versions to test against according to the product lifecycles API.")
    if len(central_chart_versions) == 0 or len(sensor_chart_versions) == 0:
        logging.info("However versions with support exceptions will still be tested against.")

    ChartVersions = namedtuple(
        "Chart_versions", ["central_version", "sensor_version"])

    # Latest central vs sensor versions in sensor_chart_versions
    test_tuples = [
        ChartVersions(central_version=latest_tag,
                      sensor_version=sensor_chart_version)
        for sensor_chart_version in sensor_chart_versions
    ]
    # Latest sensor vs central versions in central_chart_versions
    test_tuples.extend(
        [
            ChartVersions(central_version=central_chart_version,
                          sensor_version=latest_tag)
            for central_chart_version in central_chart_versions
        ]
    )

    # Currently there are no support exceptions, the last one expired on 2024-06-30, see:
    # https://issues.redhat.com/browse/ROX-18223
    # Add new support exceptions here when negotiated
    support_exceptions = []

    test_tuples.extend(
        support_exception
        for support_exception in support_exceptions
        if support_exception not in test_tuples
    )
    return test_tuples


if __name__ == "__main__":
    main()
