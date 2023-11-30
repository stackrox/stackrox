import os
import subprocess
import tempfile
from time import sleep
import unittest

from clusters import GKECluster

# pylint: disable=protected-access,too-many-public-methods

_dirname = os.path.dirname(__file__)


class TestGKECluster(unittest.TestCase):
    def setUp(self):

        os.environ.pop("TEST_PIDFILE", None)
        os.environ.pop("TEST_TERM_PIDFILE", None)

        GKECluster.PROVISION_TIMEOUT = 0.1
        GKECluster.WAIT_TIMEOUT = 0.1
        GKECluster.TEARDOWN_TIMEOUT = 0.1
        GKECluster.PROVISION_PATH = os.path.join(
            _dirname, "fixtures", "null.sh")
        GKECluster.WAIT_PATH = os.path.join(_dirname, "fixtures", "null.sh")
        GKECluster.REFRESH_PATH = os.path.join(_dirname, "fixtures", "null.sh")
        GKECluster.TEARDOWN_PATH = os.path.join(
            _dirname, "fixtures", "null.sh")

    def test_pass(self):
        GKECluster("test").provision().teardown()

    def with_a_missing_script(self):
        return os.path.join(_dirname, "fixtures", "doesnotexist.sh")

    def with_an_error_script(self):
        return os.path.join(_dirname, "fixtures", "error.sh")

    def with_a_timeout_script(self):
        return os.path.join(_dirname, "fixtures", "timeout.sh")

    def prepare_for_timeout_termination(self, tmp_dir):
        os.environ["TEST_TERM_PIDFILE"] = os.path.join(
            tmp_dir, "TEST_TERM_PIDFILE")
        return os.environ["TEST_TERM_PIDFILE"]

    def prepare_for_timeout_kill(self, tmp_dir):
        os.environ["TEST_PIDFILE"] = os.path.join(tmp_dir, "TEST_PIDFILE")
        return os.environ["TEST_PIDFILE"]

    def check_pidfile_and_process(self, pidfile):
        with open(pidfile, "r", encoding="utf8") as file:
            pid = int(file.read())
            with self.assertRaises(OSError):
                os.kill(pid, 0)

    def test_provision_nonexistant(self):
        GKECluster.PROVISION_PATH = self.with_a_missing_script()
        with self.assertRaises(FileNotFoundError):
            GKECluster("test").provision()

    def test_provision_error(self):
        GKECluster.PROVISION_PATH = self.with_an_error_script()
        with self.assertRaisesRegex(RuntimeError, "exit 1"):
            GKECluster("test").provision()

    def test_provision_timeout(self):
        GKECluster.PROVISION_PATH = self.with_a_timeout_script()
        with self.assertRaisesRegex(subprocess.TimeoutExpired, "timed out"):
            GKECluster("test").provision()

    def test_provision_timeout_terminates_script(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
            pidfile = self.prepare_for_timeout_termination(tmp_dir)
            GKECluster.PROVISION_PATH = self.with_a_timeout_script()
            with self.assertRaisesRegex(subprocess.TimeoutExpired, "timed out"):
                GKECluster("test").provision()
            self.check_pidfile_and_process(pidfile)

    def test_wait_nonexistant(self):
        GKECluster.WAIT_PATH = self.with_a_missing_script()
        with self.assertRaises(FileNotFoundError):
            GKECluster("test").provision()

    def test_wait_error(self):
        GKECluster.WAIT_PATH = self.with_an_error_script()
        with self.assertRaisesRegex(subprocess.CalledProcessError, "exit status 1"):
            GKECluster("test").provision()

    def test_wait_timeout(self):
        GKECluster.WAIT_PATH = self.with_a_timeout_script()
        with self.assertRaisesRegex(subprocess.TimeoutExpired, "timed out"):
            GKECluster("test").provision()

    def test_wait_timeout_terminates_script(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
            pidfile = self.prepare_for_timeout_kill(tmp_dir)
            GKECluster.WAIT_PATH = self.with_a_timeout_script()
            with self.assertRaisesRegex(subprocess.TimeoutExpired, "timed out"):
                GKECluster("test").provision()
            self.check_pidfile_and_process(pidfile)

    def test_refresh_nonexistant(self):
        GKECluster.REFRESH_PATH = self.with_a_missing_script()
        with self.assertRaises(FileNotFoundError):
            GKECluster("test").provision()

    def test_teardown_terminates_refresh(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
            pidfile = self.prepare_for_timeout_kill(tmp_dir)
            GKECluster.REFRESH_PATH = self.with_a_timeout_script()
            cluster = GKECluster("test").provision()
            while not os.path.exists(pidfile):
                sleep(0.1)
            cluster.teardown()
            self.check_pidfile_and_process(pidfile)

    def test_teardown_nonexistant(self):
        GKECluster.TEARDOWN_PATH = self.with_a_missing_script()
        with self.assertRaises(FileNotFoundError):
            GKECluster("test").provision().teardown()

    def test_teardown_error(self):
        GKECluster.TEARDOWN_PATH = self.with_an_error_script()
        with self.assertRaisesRegex(subprocess.CalledProcessError, "exit status 1"):
            GKECluster("test").provision().teardown()

    def test_teardown_timeout(self):
        GKECluster.TEARDOWN_PATH = self.with_a_timeout_script()
        with self.assertRaisesRegex(subprocess.TimeoutExpired, "timed out"):
            GKECluster("test").provision().teardown()

    def test_teardown_timeout_terminates_script(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
            pidfile = self.prepare_for_timeout_kill(tmp_dir)
            GKECluster.TEARDOWN_PATH = self.with_a_timeout_script()
            with self.assertRaisesRegex(subprocess.TimeoutExpired, "timed out"):
                GKECluster("test").provision().teardown()
            self.check_pidfile_and_process(pidfile)
