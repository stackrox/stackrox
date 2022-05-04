import unittest
from unittest.mock import Mock
from runners import ClusterTestRunner


class TestClusterTestRunner(unittest.TestCase):
    def test_provisions(self):
        cluster = Mock()
        ClusterTestRunner(cluster=cluster).run()
        cluster.provision.assert_called_once()

    def test_runs_pre_test(self):
        pre_test = Mock()
        ClusterTestRunner(pre_test=pre_test).run()
        pre_test.run.assert_called_once()

    def test_runs_test(self):
        test = Mock()
        ClusterTestRunner(test=test).run()
        test.run.assert_called_once()

    def test_tearsdown(self):
        cluster = Mock()
        ClusterTestRunner(cluster=cluster).run()
        cluster.teardown.assert_called_once()

    def test_runs_post(self):
        post = Mock()
        ClusterTestRunner(post=post).run()
        post.run.assert_called_once()

    def test_provision_failure(self):
        cluster = Mock()
        test = Mock()
        post = Mock()
        cluster.provision.side_effect = Exception("oops")
        with self.assertRaisesRegex(Exception, "oops"):
            ClusterTestRunner(cluster=cluster, test=test, post=post).run()
        test.run.assert_not_called()  # skips test
        post.run.assert_not_called()  # skips post
        cluster.teardown.assert_called_once()  # still tearsdown

    def test_pre_test_failure(self):
        cluster = Mock()
        pre_test = Mock()
        test = Mock()
        post = Mock()
        pre_test.run.side_effect = Exception("oops")
        with self.assertRaisesRegex(Exception, "oops"):
            ClusterTestRunner(
                cluster=cluster, pre_test=pre_test, test=test, post=post
            ).run()
        test.run.assert_not_called()  # skips test
        post.run.assert_not_called()  # skips post
        cluster.teardown.assert_called_once()  # still tearsdown

    def test_run_failure(self):
        cluster = Mock()
        test = Mock()
        post = Mock()
        test.run.side_effect = Exception("oops")
        with self.assertRaisesRegex(Exception, "oops"):
            ClusterTestRunner(cluster=cluster, test=test, post=post).run()
        test.run.assert_called_once()  # skips test
        post.run.assert_called_once()  # still posts
        cluster.teardown.assert_called_once()  # still tearsdown

    def test_post_failure(self):
        cluster = Mock()
        test = Mock()
        post = Mock()
        post.run.side_effect = Exception("oops")
        with self.assertRaisesRegex(Exception, "oops"):
            ClusterTestRunner(cluster=cluster, test=test, post=post).run()
        cluster.teardown.assert_called_once()  # still tearsdown

    def test_run_and_post_failure(self):
        cluster = Mock()
        test = Mock()
        post = Mock()
        test.run.side_effect = Exception("run oops")
        post.run.side_effect = Exception("post oops")
        with self.assertRaisesRegex(Exception, "run oops"):  # the run error is #1
            ClusterTestRunner(cluster=cluster, test=test, post=post).run()
        cluster.teardown.assert_called_once()  # still tearsdown

    def test_run_and_post_and_teardown_failure(self):
        cluster = Mock()
        test = Mock()
        post = Mock()
        test.run.side_effect = Exception("run oops")
        post.run.side_effect = Exception("post oops")
        cluster.teardown.side_effect = Exception("teardown oops")
        with self.assertRaisesRegex(Exception, "run oops"):  # the run error is #1
            ClusterTestRunner(cluster=cluster, test=test, post=post).run()

    def test_post_and_teardown_failure(self):
        cluster = Mock()
        test = Mock()
        post = Mock()
        post.run.side_effect = Exception("post oops")
        cluster.teardown.side_effect = Exception("teardown oops")
        with self.assertRaisesRegex(Exception, "post oops"):  # the post error is #1
            ClusterTestRunner(cluster=cluster, test=test, post=post).run()

    def test_provision_and_teardown_failure(self):
        cluster = Mock()
        test = Mock()
        post = Mock()
        cluster.provision.side_effect = Exception("provision oops")
        cluster.teardown.side_effect = Exception("teardown oops")
        with self.assertRaisesRegex(
            Exception, "provision oops"
        ):  # the provision error is #1
            ClusterTestRunner(cluster=cluster, test=test, post=post).run()

    def test_pre_test_and_teardown_failure(self):
        cluster = Mock()
        pre_test = Mock()
        test = Mock()
        post = Mock()
        pre_test.run.side_effect = Exception("pre_test oops")
        cluster.teardown.side_effect = Exception("teardown oops")
        with self.assertRaisesRegex(
            Exception, "pre_test oops"
        ):  # the pre_test error is #1
            ClusterTestRunner(
                cluster=cluster, pre_test=pre_test, test=test, post=post
            ).run()
