"""
PreTests - something to run before test but after resource provisioning.
"""

import subprocess


class NullPreTest:
    def run(self):
        pass


class PreSystemTests:
    """
    PreSystemTests - System tests (upgrade, e2e) need images.
    """

    POLL_TIMEOUT = 45 * 60

    def run(self):
        subprocess.run(
            [
                "scripts/ci/lib.sh",
                "poll_for_system_test_images",
                str(PreSystemTests.POLL_TIMEOUT),
            ],
            check=True,
            timeout=PreSystemTests.POLL_TIMEOUT * 1.2,
        )
