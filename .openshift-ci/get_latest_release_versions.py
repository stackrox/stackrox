#!/usr/bin/env python3

"""
Looks up N previous releases before the current version (or code revision) and outputs a Helm chart version for the most
recent patch for each found release.
"""

import json
import logging
import pathlib
import re
import subprocess
import sys

this_script_dir = pathlib.Path(__file__).parent
repo_root = this_script_dir.parent

helm_repo_name = "temp-stackrox-oss-repo-should-not-see-me"

add_repo_cmd = f"helm repo add {helm_repo_name} https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource"
update_repo_cmd = "helm repo update"
search_cmd = f"helm search repo {helm_repo_name} -l -o json"
remove_repo_cmd = f"helm repo remove {helm_repo_name}"

make_tag_cmd = ["make", "-C", repo_root, "--quiet", "--no-print-directory", "tag"]
get_previous_y_stream_cmd = repo_root / "scripts/get-previous-y-stream.sh"


def main(argv):
    logging.basicConfig(stream=sys.stderr, level=logging.DEBUG)
    n = int(argv[1]) if len(argv) > 1 else 4
    helm_versions = get_previously_released_helm_chart_versions("stackrox-secured-cluster-services", n)
    logging.info(f"Previous {n} helm chart versions released:")
    print("\n".join(helm_versions))


def get_previously_released_helm_chart_versions(chart_name, num_versions):
    """
    Looks up current tag (with `make tag`), resolves `num_versions` existing releases preceding the current tag, for
    each release finds the most recently available patch version, returns helm chart version for the most recent patch
    version for each identified release.
    All this makes possible to deploy most recent patches of previous releases with Helm in version compatibility tests.
    """
    add_helm_repo()
    try:
        update_helm_repo()
        return __get_previously_released_helm_chart_versions(chart_name, num_versions)
    finally:
        remove_helm_repo()


def __get_previously_released_helm_chart_versions(chart_name, num_versions):
    charts = read_charts()
    logging.info(f"Discovered {len(charts)} charts")

    current_tag = get_current_tag()
    logging.info(f"Current tag: {current_tag}")

    candidate_releases = get_n_candidate_releases(num_versions, current_tag)
    logging.info(f"Candidate releases to look for: {candidate_releases}")

    latest_charts = get_latest_charts_for_releases(charts, chart_name, candidate_releases)
    logging.debug(f"Identified these charts as latest for requested releases: {latest_charts}")

    return [c["version"] for c in latest_charts]


def read_charts():
    json_str = run_command(search_cmd, log_stdout=False)
    charts_from_json = json.loads(json_str)

    charts_in_repo = [c for c in charts_from_json if c["name"].startswith(helm_repo_name + "/")]

    release_charts = [c for c in charts_in_repo if is_release_version(c["app_version"])]

    for entry in release_charts:
        entry["name_without_repo"] = entry["name"].partition("/")[2]

    for entry in release_charts:
        entry["parsed_app_version"] = parse_version(entry["app_version"])

    return release_charts


def is_release_version(version):
    return re.search(r"^\d+\.\d+\.\d+$", version) is not None


def parse_version(version_str):
    nums = [int(s) for s in version_str.split(".")]
    return {"major": nums[0], "minor": nums[1], "patch": nums[2]}


def get_current_tag():
    return run_command(make_tag_cmd, shell=False).strip()


def get_n_candidate_releases(count, starting_version):
    # get_previous_y_stream_cmd used below does not output the most recent release that happened; it outputs the
    # _previous_ release. The effect is that for a development version like 4.0.x-261-ge63c3e6591-dirty, the `for` loop
    # below will produce 3.74.0, 3.73.0, ...
    # However, tagging/versioning in the stackrox repo is done in such a way that development builds get tag 4.0.x-...
    # once the release 4.0.0 gets started. Therefore, we can, and have to, use the current tag to detect the most recent
    # release (4.0.0 in this example).
    # Note, that the most recent release may not be completed yet and so helm charts won't be found for it. This should
    # be handled ok by the rest of this script.
    result = [make_current_release_version(starting_version)]

    v = starting_version
    for i in range(0, count - 1):
        v = run_command([get_previous_y_stream_cmd, v], shell=False).strip()
        result.append(parse_version(v))

    return result


def make_current_release_version(current_tag):
    """
    Convert development tag like 4.0.x-261-ge63c3e6591-dirty to a release version like 4.0.0
    """
    x, y = [int(s) for s in current_tag.split(".")[0:2]]
    return {"major": x, "minor": y, "patch": 0}


def get_latest_charts_for_releases(charts, chart_name, release_versions):
    result = []
    for v in release_versions:
        c = get_sorted_charts_for_release(charts, chart_name, v)
        if c:
            result.append(c[0])
    return result


def get_sorted_charts_for_release(charts, chart_name, release_version):
    candidates = []

    for c in charts:
        if c["name_without_repo"] == chart_name and \
                c["parsed_app_version"]["major"] == release_version["major"] and \
                c["parsed_app_version"]["minor"] == release_version["minor"]:
            candidates.append(c)

    return sorted(candidates, key=lambda x: x["parsed_app_version"]["patch"], reverse=True)


def add_helm_repo():
    logging.info("Adding temp helm repository...")
    run_command(add_repo_cmd)


def update_helm_repo():
    logging.info("Updating temp helm repository...")
    run_command(update_repo_cmd)


def remove_helm_repo():
    logging.info("Removing temp helm repository...")
    run_command(remove_repo_cmd)


def run_command(command, shell=True, log_stdout=True):
    result = subprocess.run(command, shell=shell, encoding='utf-8',
                            stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    stdout = format_command_output("Stdout", result.stdout) if log_stdout else ""
    stderr = format_command_output("Stderr", result.stderr)
    logging.debug(f"Got exit code {result.returncode} for command: {command}{stdout}{stderr}")

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
