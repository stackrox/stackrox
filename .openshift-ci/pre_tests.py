"""
PreTests - something to run before test but after resource provisioning.
"""

import subprocess


class NullPreTest:
    def run(self):
        pass


class PreSystemTests:
    """
    PreSystemTests - System tests (upgrade, e2e) need images and have a target
    cluster from which to get information.
    """

    VERSIONS_TIMEOUT = 10 * 60
    POLL_TIMEOUT = 60 * 60

    def run(self):
        subprocess.run(
            [
                "scripts/ci/lib.sh",
                "gather_debug_for_cluster_under_test",
            ],
            check=False,
            timeout=PreSystemTests.VERSIONS_TIMEOUT,
        )

        subprocess.run(
            [
                "scripts/ci/lib.sh",
                "poll_for_system_test_images",
                str(PreSystemTests.POLL_TIMEOUT),
            ],
            check=True,
            timeout=PreSystemTests.POLL_TIMEOUT * 1.2,
        )
