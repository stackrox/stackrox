#!/usr/bin/env python3

"""
Common test run patterns
"""

from datetime import datetime
from clusters import NullCluster
from pre_tests import NullPreTest
from ci_tests import NullTest
from post_tests import NullPostTest


class TestSet:
    """Represents a test set, i.e. a test with pre-test and post-test logic."""

    def __init__(self, name, pre=None, test=None, post=None, always_run=True):
        self._name = name
        self._pre_test = pre if pre else NullPreTest()
        self._actual_test = test if test else NullTest()
        self._post_test = post if post else NullPostTest()
        self._always_run = always_run

    @property
    def name(self):
        return self._name

    @property
    def always_run(self):
        return self._always_run

    def run(self):
        exception = None
        try:
            log_event("About to run pre test", self)
            self._pre_test.run()
            log_event("pre test completed", self)
        except Exception as err:
            log_event(f"ERROR: pre test failed [{err}]", self)
            exception = err
        if exception is None:
            try:
                log_event("About to run test", self)
                self._actual_test.run()
                log_event("test completed", self)
            except Exception as err:
                log_event(f"ERROR: test failed [{err}]", self)
                exception = err
            try:
                log_event("About to run post test", self)
                self._post_test.run(
                    test_outputs=self._actual_test.test_outputs,
                )
                log_event("post test completed", self)
            except Exception as err:
                log_event(f"ERROR: post test failed [{err}]", self)
                if exception is None:
                    exception = err

        if exception is not None:
            raise exception


class ClusterTestSetsRunner:
    """A cluster test runner that runs multiple sets of pre, test & post steps
    wrapped by a cluster provision and with similar semantics to
    ClusterTestRunner. Each test set will attempt to run regardless of the outcome of
    prior sets. This can be overridden on a set by set basis with 'always_run'"""

    def __init__(
        self,
        cluster=NullCluster(),
        initial_pre_test=NullPreTest(),
        sets=None,
        final_post=NullPostTest(),
    ):
        self.cluster = cluster
        self.initial_pre_test = initial_pre_test
        if sets is None:
            sets = []
        self.sets = sets
        self.final_post = final_post

    def run(self):
        exception = self.cluster_provision()
        exception = self.run_initial_pre_test(exception)

        if exception is None:
            for test_set in self.sets:
                if exception is None or test_set.always_run:
                    try:
                        log_event("About to run", test_set)
                        test_set.run()
                        log_event("run completed", test_set)
                    except Exception as err:
                        log_event(f"ERROR: run failed [{err}]", test_set)
                        if exception is None:
                            exception = err

        try:
            log_event("About to teardown")
            self.cluster.teardown()
            log_event("teardown completed")
        except Exception as err:
            log_event(f"ERROR: teardown failed [{err}]")
            if exception is None:
                exception = err

        try:
            log_event("About to run final post")
            self.final_post.run()
            log_event("final post completed")
        except Exception as err:
            log_event(f"ERROR: final post failed [{err}]")
            if exception is None:
                exception = err

        if exception is not None:
            raise exception

    def cluster_provision(self):
        exception = None
        try:
            log_event("About to provision")
            self.cluster.provision()
            log_event("provisioned")
        except Exception as err:
            log_event(f"ERROR: provision failed [{err}]")
            exception = err
        return exception

    def run_initial_pre_test(self, exception):
        if exception is None:
            try:
                log_event("About to run initial pre test")
                self.initial_pre_test.run()
                log_event("initial pre test completed")
            except Exception as err:
                log_event(f"ERROR: initial pre test failed [{err}]")
                exception = err
        return exception


def log_event(msg, test_set=None):
    now = datetime.now()
    time = now.strftime("%H:%M:%S")
    marker = "****"
    if test_set and test_set.name:
        msg = f"{msg} [{test_set.name}]"
    print(marker)
    print(f"{marker} {time}: {msg}")
    print(marker)


# pylint: disable=too-many-arguments
class ClusterTestRunner(ClusterTestSetsRunner):
    """A simple cluster test runner that:
    . provisions a cluster
    . runs any pre_test (if provision was successful)
    . runs the test (if provisioned and any pre_test was successful)
    . runs post_test (if the test ran)
    . tears down the cluster"""

    def __init__(
        self,
        cluster=NullCluster(),
        pre_test=NullPreTest(),
        test=NullTest(),
        post_test=NullPostTest(),
        final_post=NullPostTest(),
    ):
        super().__init__(
            cluster=cluster,
            final_post=final_post,
            sets=[TestSet(name=None, pre=pre_test, test=test, post=post_test)],
        )
