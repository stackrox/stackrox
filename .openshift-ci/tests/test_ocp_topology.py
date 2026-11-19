"""Exercise topology selection without credentials or cloud provisioning."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class TestOcpTopology(unittest.TestCase):
    def test_topologies(self):
        cases = {
            "specialized": ("vm-scanning-e2e-tests", "n2-standard-8"),
            "compliance": ("compliance-e2e-tests", "e2-standard-8"),
            "operator": ("operator-e2e-tests", "e2-standard-8"),
            "nongroovy": ("nongroovy-e2e-tests", "e2-standard-8"),
            "scanner": ("scanner-v4-install-tests", "e2-standard-8"),
            "qa": ("qa-e2e-tests", "e2-standard-8"),
            "ui": ("ui-e2e-tests", "e2-standard-8"),
            "unknown": ("unknown-e2e-tests", "e2-standard-8"),
        }
        script = Path(__file__).resolve().parents[1] / "ocp-topology.sh"
        for name, (suite, worker_type) in cases.items():
            for version in ("4-22", "4-12", "dev-preview", "5-0"):
                with self.subTest(name=name, version=version):
                    env = {"PATH": os.environ["PATH"]}
                    env["JOB_NAME"] = (
                        "pull-ci-stackrox-stackrox-master-ocp-"
                        f"{version}-{suite}")
                    with tempfile.TemporaryDirectory() as directory:
                        env["SHARED_DIR"] = directory
                        result = subprocess.run(
                            ["bash", "-eu", "-c", '''
set_ci_shared_export() {
    export "$1=$2"
    printf 'export %s=%q\\n' "$1" "$2" >> "$SHARED_DIR/shared_env"
}
info() { :; }
die() { echo "$*" >&2; return 1; }
source "$1"
configure_ocp_topology
cat "$SHARED_DIR/shared_env"
''', "test", str(script)], env=env,
                            capture_output=True, text=True, check=False)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    sno = version == "4-22" and name in (
                        "operator", "nongroovy", "scanner", "qa", "ui")
                    self.assertIn(
                        f"WORKER_NODE_COUNT={0 if sno else 2}", result.stdout)
                    self.assertIn(
                        f"WORKER_NODE_TYPE={worker_type}", result.stdout)
                    if sno:
                        self.assertIn("MASTER_NODE_COUNT=1", result.stdout)
                        self.assertIn("MASTER_NODE_TYPE=e2-standard-16",
                                      result.stdout)
                    else:
                        self.assertNotIn("MASTER_NODE_COUNT=", result.stdout)


if __name__ == "__main__":
    unittest.main()
