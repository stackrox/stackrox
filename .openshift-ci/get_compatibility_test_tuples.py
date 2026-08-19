#!/usr/bin/env python3

"""
Returns the central-sensor version tuples to be used for compatibility testing.
"""
import logging
import re
import subprocess
import sys
from collections import namedtuple
from pathlib import Path

from get_latest_helm_chart_versions import (
    get_supported_helm_chart_versions,
)


# We run compatibility tests against supported older versions of Stackrox.
# get_compatibility_test_tuples.py provides the function get_compatibility_test_tuples() which is called in our
# compatibility tests that returns all tuples of central and sensor versions for which:
#    1. Central is latest and Sensor is a supported older version OR
#    2. Sensor is latest and Central is a supported older version
# These are returned only if charts for both versions could be found.
# After running get_compatibility_test_tuples.py I received the following output:
# INFO:root:Listing supported versions tuples:
# INFO:root:Tuple 1: {Central v4.12.x-793-gf6bf3cc40e - Sensor v4.11.2}
# INFO:root:Tuple 2: {Central v4.12.x-793-gf6bf3cc40e - Sensor v4.10.6}
# INFO:root:Tuple 3: {Central v4.12.x-793-gf6bf3cc40e - Sensor v4.9.10}
# INFO:root:Tuple 4: {Central v4.11.2 - Sensor v4.12.x-793-gf6bf3cc40e}
# INFO:root:Tuple 5: {Central v4.10.6 - Sensor v4.12.x-793-gf6bf3cc40e}
# INFO:root:Tuple 6: {Central v4.9.10 - Sensor v4.12.x-793-gf6bf3cc40e}
# If no supported versions with available charts are found, an empty list is returned.
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


# Returns True if the product_version is newer than the current_version.
#
# current_version examples: '4.11.x-736-g48077a980e', '4.11.1-rc.2', '4.11.1'
# product_version examples:  '4.9.3', '4.11.0'
def is_newer_version(current_version: str, product_version: str):
    product_parts = [int(x) for x in product_version.split('.')]

    # Parse the numeric major.minor[.patch] prefix from current_version, stopping
    # at any component that does not begin with a digit (e.g. 'x-736-ghash').
    # This correctly handles:
    #   dev:     '4.11.x-736-ghash' -> [4, 11]
    #   rc:      '4.11.1-rc.2'      -> [4, 11, 1]
    #   release: '4.11.1'           -> [4, 11, 1]
    current_parts = []
    for part in current_version.split('.')[:3]:
        m = re.match(r'^(\d+)', part)
        if not m:
            break
        current_parts.append(int(m.group(1)))

    for (current, product) in zip(current_parts, product_parts):
        if current > product:
            return False
        if current < product:
            return True

    return False


def get_compatibility_test_tuples():
    # start logging
    logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

    central_versions, sensor_versions = get_supported_helm_chart_versions()

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
    central_versions = [i for i in central_versions
                        if not is_newer_version(current_version=latest_tag,
                                                product_version=i)]
    sensor_versions = [i for i in sensor_versions
                       if not is_newer_version(current_version=latest_tag,
                                               product_version=i)]

    if len(central_versions) == 0:
        logging.info("Found no older central versions to test against according to the product lifecycles API.")
    if len(sensor_versions) == 0:
        logging.info("Found no older sensor versions to test against according to the product lifecycles API.")

    VersionTuple = namedtuple("VersionTuple", ["central_version", "sensor_version"])

    # Latest central vs older sensor versions
    test_tuples = [
        VersionTuple(central_version=latest_tag,
                     sensor_version=sensor_version)
        for sensor_version in sensor_versions
    ]
    # Older central versions vs latest sensor
    test_tuples.extend(
        [
            VersionTuple(central_version=central_version,
                         sensor_version=latest_tag)
            for central_version in central_versions
        ]
    )

    return test_tuples


if __name__ == "__main__":
    main()
