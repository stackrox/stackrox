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

    POLL_TIMEOUT = 30 * 60

    def run(self):
        subprocess.run(
            [
                "scripts/ci/lib.sh",
                "poll_for_opensource_images",
                "120",
            ],
            check=True,
            timeout=PreSystemTests.POLL_TIMEOUT * 1.2,
        )
