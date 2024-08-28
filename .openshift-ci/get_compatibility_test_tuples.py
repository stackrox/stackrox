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


def main():
    logging.basicConfig(stream=sys.stderr, level=logging.DEBUG)
    test_tuples = get_compatibility_test_tuples()
    logging.info(
        f"Listing supported versions tuples:"
    )
    i = 0
    for test_tuple in test_tuples:
        i += 1
        logging.info(
            f"Tuple {i}: {{Central v{test_tuple.central_version} - Sensor v{test_tuple.sensor_version}}}"
        )


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
    # however a new support exception is being negotiated, add it here when it's ready
    support_exceptions = [
        ChartVersions(
            central_version=latest_tag,
            sensor_version=get_latest_helm_chart_version_for_specific_release(
                "stackrox-secured-cluster-services", Release(major=3, minor=74)
            ),
        )
    ]

    test_tuples.extend(
        support_exception
        for support_exception in support_exceptions
        if support_exception not in test_tuples
    )
    return test_tuples


if __name__ == "__main__":
    main()
