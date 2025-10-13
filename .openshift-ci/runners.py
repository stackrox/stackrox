#!/usr/bin/env python3

"""
Common test run patterns
"""

from datetime import datetime
from clusters import NullCluster
from pre_tests import NullPreTest
from ci_tests import NullTest
from post_tests import NullPostTest


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
            for idx, test_set in enumerate(self.sets):
                test_set = {
                    **{
                        "name": f"set {idx + 1}",
                        "pre_test": NullPreTest(),
                        "test": NullTest(),
                        "post_test": NullPostTest(),
                        "always_run": True,
                    },
                    **test_set,
                }
                if exception is None or test_set["always_run"]:
                    try:
                        self.log_event("About to run", test_set)
                        self.run_test_set(test_set)
                        self.log_event("run completed", test_set)
                    except Exception as err:
                        self.log_event(f"ERROR: run failed [{err}]", test_set)
                        if exception is None:
                            exception = err

        try:
            self.log_event("About to teardown")
            self.cluster.teardown()
            self.log_event("teardown completed")
        except Exception as err:
            self.log_event(f"ERROR: teardown failed [{err}]")
            if exception is None:
                exception = err

        try:
            self.log_event("About to run final post")
            self.final_post.run()
            self.log_event("final post completed")
        except Exception as err:
            self.log_event(f"ERROR: final post failed [{err}]")
            if exception is None:
                exception = err

        if exception is not None:
            raise exception

    def cluster_provision(self):
        exception = None
        try:
            self.log_event("About to provision")
            self.cluster.provision()
            self.log_event("provisioned")
        except Exception as err:
            self.log_event(f"ERROR: provision failed [{err}]")
            exception = err
        return exception

    def run_initial_pre_test(self, exception):
        if exception is None:
            try:
                self.log_event("About to run initial pre test")
                self.initial_pre_test.run()
                self.log_event("initial pre test completed")
            except Exception as err:
                self.log_event(f"ERROR: initial pre test failed [{err}]")
                exception = err
        return exception

    def run_test_set(self, test_set):
        exception = None
        try:
            self.log_event("About to run pre test", test_set)
            test_set["pre_test"].run()
            self.log_event("pre test completed", test_set)
        except Exception as err:
            self.log_event(f"ERROR: pre test failed [{err}]", test_set)
            exception = err
        if exception is None:
            try:
                self.log_event("About to run test", test_set)
                test_set["test"].run()
                self.log_event("test completed", test_set)
            except Exception as err:
                self.log_event(f"ERROR: test failed [{err}]", test_set)
                exception = err
            try:
                self.log_event("About to run post test", test_set)
                test_set["post_test"].run(
                    test_outputs=test_set["test"].test_outputs,
                )
                self.log_event("post test completed", test_set)
            except Exception as err:
                self.log_event(f"ERROR: post test failed [{err}]", test_set)
                if exception is None:
                    exception = err

        if exception is not None:
            raise exception

    def log_event(self, msg, test_set=None):
        now = datetime.now()
        time = now.strftime("%H:%M:%S")
        marker = "****"
        if test_set is not None and test_set["name"] is not None:
            msg = f"{msg} [{test_set['name']}]"
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
            sets=[
                {
                    "name": None,
                    "pre_test": pre_test,
                    "test": test,
                    "post_test": post_test,
                }
            ],
        )
