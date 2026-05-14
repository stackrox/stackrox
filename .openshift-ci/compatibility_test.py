#!/usr/bin/env -S python3 -u

"""
Runs version compatibility tests against the supplied testfunc
"""
import logging
import os
import shutil
import subprocess
import sys
import tempfile

from pre_tests import (
    PreSystemTests,
    CollectionMethodOverridePreTest
)
from post_tests import PostClusterTest, FinalPost
from runners import ClusterTestSetsRunner, TestSet
from clusters import GKECluster
from get_compatibility_test_tuples import (
    get_compatibility_test_tuples,
)

HELM_REPO_NAME = "tmp-srox-compat"
HELM_CHARTS_GIT_REPO = "https://github.com/stackrox/helm-charts.git"
HELM_CHART_URL_FALLBACK = "https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource"


def _clone_helm_charts_repo():
    """Shallow-clone the helm-charts repo and return (clone_dir, file:// URL).

    Uses GITHUB_TOKEN for authenticated access when available.
    Returns (None, fallback_url) on failure.
    """
    clone_dir = tempfile.mkdtemp(prefix="helm-charts-clone-")
    repo_url = HELM_CHARTS_GIT_REPO
    token = os.environ.get("GITHUB_TOKEN", "")
    if token:
        repo_url = f"https://x-access-token:{token}@github.com/stackrox/helm-charts.git"
    try:
        subprocess.run(
            ["git", "clone", "--depth", "1", repo_url, clone_dir],
            check=True, timeout=120,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        )
        chart_url = f"file://{clone_dir}/opensource"
        logging.info("Cloned helm-charts repo to %s", clone_dir)
        return clone_dir, chart_url
    except Exception:
        logging.warning("Failed to clone helm-charts repo, falling back to remote URL", exc_info=True)
        shutil.rmtree(clone_dir, ignore_errors=True)
        return None, HELM_CHART_URL_FALLBACK


def run_compatibility_tests(testfunc, cluster_name):
    # start logging
    logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

    clone_dir, chart_url = _clone_helm_charts_repo()
    os.environ["COMPAT_HELM_CHART_URL"] = chart_url
    try:
        # Get the test tuples (central_version, sensor_version) for supported versions with available helm charts
        test_tuples = get_compatibility_test_tuples()

        if len(test_tuples) > 0:
            subprocess.run(
                ["helm", "repo", "add", HELM_REPO_NAME, chart_url],
                check=True, timeout=120,
            )
            subprocess.run(
                ["helm", "repo", "update", HELM_REPO_NAME],
                check=True, timeout=120,
            )
            os.environ["COMPAT_HELM_REPO_NAME"] = HELM_REPO_NAME
            try:
                sets = []
                for test_tuple in test_tuples:
                    os.environ["ROX_TELEMETRY_STORAGE_KEY_V1"] = 'DISABLED'
                    test_versions = f'{test_tuple.central_version}--{test_tuple.sensor_version}'

                    # expected version string is like 74.x.x for ACS 3.74 versions
                    is_3_74_sensor = test_tuple.sensor_version.startswith('74')

                    logging.info("Running compatibility tests for central-v%s, sensor-v%s with function %s",
                                 test_tuple.central_version, test_tuple.sensor_version, testfunc.__name__)

                    sets.append(
                        TestSet(
                            f'version compatibility tests: {test_versions}',
                            test=testfunc(test_tuple.central_version, test_tuple.sensor_version),
                            post=PostClusterTest(
                                collect_collector_metrics=not is_3_74_sensor,
                                check_stackrox_logs=True,
                                artifact_destination_prefix=test_versions,
                            ),
                            # Collection not supported on 3.74
                            pre=CollectionMethodOverridePreTest("NO_COLLECTION" if is_3_74_sensor else "core_bpf")
                        )
                    )
                ClusterTestSetsRunner(
                    cluster=GKECluster(cluster_name,
                                       machine_type="e2-standard-8", num_nodes=2),
                    initial_pre_test=PreSystemTests(),
                    sets=sets,
                    final_post=FinalPost(
                        store_qa_tests_data=True,
                    ),
                ).run()
            finally:
                subprocess.run(
                    ["helm", "repo", "remove", HELM_REPO_NAME],
                    check=False, timeout=30,
                )
                os.environ.pop("COMPAT_HELM_REPO_NAME", None)
        else:
            logging.info("There are currently no supported older versions or support exceptions that require compatibility "
                         "testing.")
    finally:
        os.environ.pop("COMPAT_HELM_CHART_URL", None)
        if clone_dir:
            shutil.rmtree(clone_dir, ignore_errors=True)
