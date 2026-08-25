#!/usr/bin/env python3

"""
Looks up supported releases and outputs a product version for the most
recent patch for each found release.
"""

# pylint: disable=logging-fstring-interpolation

import json
import logging
import pathlib
import re
import subprocess
import sys
from urllib.error import URLError
from urllib.request import Request, urlopen

from collections import namedtuple

this_script_dir = pathlib.Path(__file__).parent
repo_root = this_script_dir.parent

HELM_REPO_NAME = "temp-stackrox-oss-repo-should-not-see-me"

ADD_REPO_CMD = f"""helm repo add {HELM_REPO_NAME} \
https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource"""
UPDATE_REPO_CMD = "helm repo update"
SEARCH_CMD = f"helm search repo {HELM_REPO_NAME} --versions --output json"
REMOVE_REPO_CMD = f"helm repo remove {HELM_REPO_NAME}"

Version = namedtuple("Version", ["major", "minor", "patch"])

# Here we call "release" (or Y-Stream) the first appearance of X.Y.0 version.
Release = namedtuple("Release", ["major", "minor"])

# API from which we can gather which versions of Stackrox are currently supported
PRODUCT_LIFECYCLES_API = "https://access.redhat.com/product-life-cycles/api/v1/products?name=" \
                         "Red%20Hat%20Advanced%20Cluster%20Security%20for%20Kubernetes"

# Default value of N, the number of previous releases to look up.
# The current release cadence is 9 weeks (sometimes extended but not reduced),
# i.e. 9*7=63 days.
# The current support period is 6 months, i.e. at most 184 days.
# Therefore, at most 3 releases will be in support at any given moment of time
# with the current cadence and support period.
NUM_RELEASES_DEFAULT = 3

# From this release onward the stackrox-operator chart is used instead of the
# per-component legacy charts (stackrox-central-services,
# stackrox-secured-cluster-services), which are not published for new releases.
# Drop legacy chart inspection can be dropped, once 4.10 goes EOL.
OPERATOR_CHART_FIRST_RELEASE = Release(major=4, minor=11)

OPERATOR_CHART_NAME = "stackrox-operator"


def main(argv):
    logging.basicConfig(stream=sys.stderr, level=logging.DEBUG)
    num_releases = int(argv[1]) if len(argv) > 1 else NUM_RELEASES_DEFAULT
    helm_versions = get_latest_helm_chart_versions(
        "stackrox-secured-cluster-services", num_releases
    )
    logging.info(
        f"Product versions for the latest {num_releases} releases:")

    logging.info("\n".join(helm_versions))
    supported_versions_string = [[f"{version.major}.{version.minor}"] for version in get_supported_releases()]
    supported_central_versions_from_api, supported_sensor_versions_from_api = get_supported_helm_chart_versions()
    logging.info(
        f"\nThe product lifecycles API denotes support for the following versions: {supported_versions_string}\n"
        f"Found product versions for the following supported versions: "
        f"central{supported_central_versions_from_api} - sensor{supported_sensor_versions_from_api}"
    )


def _is_legacy_release(release: Release) -> bool:
    """Returns True for releases that use per-component charts (pre-4.11)."""
    return release < OPERATOR_CHART_FIRST_RELEASE


def _chart_to_product_version(chart: dict) -> str:
    """Return a product version string (e.g. '4.9.3') from a chart dict."""
    av = chart["parsed_app_version"]
    return f"{av.major}.{av.minor}.{av.patch}"


def get_latest_helm_chart_versions(chart_name, num_releases=NUM_RELEASES_DEFAULT):
    add_helm_repo()
    try:
        update_helm_repo()
        return __get_latest_helm_chart_versions(chart_name, num_releases)
    finally:
        remove_helm_repo()


def get_latest_helm_chart_version_for_specific_release(chart_name, release):
    add_helm_repo()
    try:
        update_helm_repo()
        return __get_latest_helm_chart_version_for_specific_release(chart_name, release)
    finally:
        remove_helm_repo()


def get_supported_helm_chart_versions():
    add_helm_repo()
    try:
        update_helm_repo()
        return __get_supported_helm_chart_versions()
    finally:
        remove_helm_repo()


def __get_supported_helm_chart_versions():
    supported_central_versions = []
    supported_sensor_versions = []

    charts = read_charts()  # one helm search, reused for all releases

    supported_releases = get_supported_releases()
    for release in supported_releases:
        if _is_legacy_release(release):
            # Pre-4.11: separate charts for central and sensor.
            for chart_name, version_list in [
                ("stackrox-central-services", supported_central_versions),
                ("stackrox-secured-cluster-services", supported_sensor_versions),
            ]:
                filtered = filter_charts_by_name(charts, chart_name)
                if any(version_to_release(c["parsed_app_version"]) == release
                       for c in filtered):
                    chart = get_latest_chart_for_specific_release(filtered, release)
                    version_list.append(_chart_to_product_version(chart))
                else:
                    logging.debug(
                        f"Supported version \"{release.major}.{release.minor}\" has no "
                        f"corresponding chart for {chart_name}."
                    )
        else:
            # 4.11+: one operator chart covers both central and sensor.
            filtered = filter_charts_by_name(charts, OPERATOR_CHART_NAME)
            if any(version_to_release(c["parsed_app_version"]) == release
                   for c in filtered):
                chart = get_latest_chart_for_specific_release(filtered, release)
                version = _chart_to_product_version(chart)
                supported_central_versions.append(version)
                supported_sensor_versions.append(version)
            else:
                logging.warning(
                    f"Supported version \"{release.major}.{release.minor}\" has no "
                    f"corresponding chart for {OPERATOR_CHART_NAME}. "
                    f"This release will be skipped in compatibility testing."
                )

    return supported_central_versions, supported_sensor_versions


def get_supported_releases():
    supported_releases = []
    data = __get_data_from_product_lifecycles_api()
    if "data" not in data or len(data["data"]) == 0 or "versions" not in data["data"][0]:
        logging.debug("Found no RHACS releases in PRODUCT_LIFECYCLES_API")
        return []
    releases = data["data"][0]["versions"]

    for release in releases:
        if "type" in release and release["type"] != "End of life":
            supported_releases.append(parse_release(release["name"]))
    return supported_releases


def __get_data_from_product_lifecycles_api():
    req = Request(
        url=PRODUCT_LIFECYCLES_API,
        headers={'User-Agent': 'Mozilla/5.0'}
    )
    try:
        with urlopen(req) as response:
            response_bytes = response.read()
            response_string = response_bytes.decode('utf-8')
            data = json.loads(response_string)
            return data
    except URLError as exception:
        logging.debug(f"Failed to open URL {PRODUCT_LIFECYCLES_API} with error:\n{repr(exception)}")
    except json.JSONDecodeError as exception:
        logging.debug(f"Failed to load JSON from API response from {PRODUCT_LIFECYCLES_API} with error:"
                      f"\n{repr(exception)}")
    except ValueError as exception:
        logging.debug(f"Failed to decode API response from {PRODUCT_LIFECYCLES_API} with error:\n{repr(exception)}")
    return []


def __get_latest_helm_chart_versions(chart_name, num_releases):
    charts = read_charts()
    logging.info(f"Discovered total {len(charts)} charts")

    filtered_charts = filter_charts_by_name(charts, chart_name)
    logging.info(
        f"Found {len(filtered_charts)} charts with the given name {chart_name}"
    )

    latest_charts = get_latest_chart_for_each_release(filtered_charts)[
        :num_releases]
    logging.debug(
        f"Identified these charts as {num_releases} latest: {latest_charts}")

    return [_chart_to_product_version(c) for c in latest_charts]


def __get_latest_helm_chart_version_for_specific_release(chart_name, release):
    charts = read_charts()
    logging.info(f"Discovered total {len(charts)} charts")

    filtered_charts = filter_charts_by_name(charts, chart_name)
    logging.info(
        f"Found {len(filtered_charts)} charts with the given name {chart_name}")

    latest_chart = get_latest_chart_for_specific_release(
        filtered_charts, release)
    logging.debug(
        f"Identified {latest_chart} as latest version of release {release}")

    return _chart_to_product_version(latest_chart)


def read_charts():
    json_str = run_command(SEARCH_CMD, log_stdout=False)
    charts_from_json = json.loads(json_str)

    release_charts = [
        c for c in charts_from_json if is_release_version(c["app_version"])
    ]

    for entry in release_charts:
        entry["parsed_app_version"] = parse_version(entry["app_version"])

    return release_charts


def is_release_version(version):
    return re.search(r"^\d+\.\d+\.\d+$", version) is not None


def parse_version(version_str):
    nums = [int(s) for s in version_str.split(".")]
    return Version(major=nums[0], minor=nums[1], patch=nums[2])


def parse_release(release_str):
    nums = [int(s) for s in release_str.split(".")]
    return Release(major=nums[0], minor=nums[1])


def filter_charts_by_name(charts, chart_name):
    return [c for c in charts if c["name"] == f"{HELM_REPO_NAME}/{chart_name}"]


def get_latest_chart_for_each_release(charts):
    sorted_charts = sorted(
        charts, key=lambda x: x["parsed_app_version"], reverse=True)

    result = []
    release = None

    for chart in sorted_charts:
        chart_release = version_to_release(chart["parsed_app_version"])
        if chart_release != release:
            result.append(chart)
            release = chart_release

    return result


def get_latest_chart_for_specific_release(charts, release):
    sorted_charts = sorted(
        charts, key=lambda x: x["parsed_app_version"], reverse=True)

    for chart in sorted_charts:
        chart_release = version_to_release(chart["parsed_app_version"])
        if chart_release == release:
            return chart

    raise RuntimeError(
        f"Could not find chart for requested release version {release}")


def version_to_release(version):
    return Release(major=version.major, minor=version.minor)


def add_helm_repo():
    logging.info("Adding temp helm repository...")
    run_command(ADD_REPO_CMD)


def update_helm_repo():
    logging.info("Updating temp helm repository...")
    run_command(UPDATE_REPO_CMD)


def remove_helm_repo():
    logging.info("Removing temp helm repository...")
    run_command(REMOVE_REPO_CMD)


def run_command(command, log_stdout=True):
    result = subprocess.run(
        command,
        shell=True,
        encoding="utf-8",
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False
    )

    stdout = format_command_output(
        "Stdout", result.stdout) if log_stdout else ""
    stderr = format_command_output("Stderr", result.stderr)
    logging.debug(
        f"Got exit code {result.returncode} for command: {command}{stdout}{stderr}"
    )

    result.check_returncode()

    return result.stdout


def format_command_output(name, output):
    out_no_trailing_newline = output.rstrip()
    if not out_no_trailing_newline:
        return ""
    prefix = "\n" if len(out_no_trailing_newline.splitlines()) > 1 else " "
    return f"\n{name}:{prefix}{out_no_trailing_newline}"


if __name__ == "__main__":
    main(sys.argv)
