#!/usr/bin/env python3

"""
Common test run patterns
"""

from datetime import datetime
from clusters import NullCluster
from pre_tests import NullPreTest
from ci_tests import NullTest
from posts import NullPost


class ClusterTestRunner:
    def __init__(
        self,
        cluster=NullCluster(),
        pre_test=NullPreTest(),
        test=NullTest(),
        post=NullPost(),
    ):
        self.cluster = cluster
        self.pre_test = pre_test
        self.test = test
        self.post = post

    def run(self):
        hold = None
        try:
            self.log_significant_event("About to provision")
            self.cluster.provision()
            self.log_significant_event("provisioned")
        except Exception as err:
            self.log_significant_event("provision failed")
            hold = err
        if hold is None:
            try:
                self.log_significant_event("About to run pre test")
                self.pre_test.run()
                self.log_significant_event("pre test completed")
            except Exception as err:
                self.log_significant_event("pre test failed")
                hold = err
        if hold is None:
            try:
                self.log_significant_event("About to run test")
                self.test.run()
                self.log_significant_event("test completed")
            except Exception as err:
                self.log_significant_event("test failed")
                hold = err
            try:
                self.log_significant_event("About to post")
                self.post.run(test_output_dirs=self.test.test_output_dirs)
                self.log_significant_event("post completed")
            except Exception as err:
                self.log_significant_event("post failed")
                if hold is None:
                    hold = err

        try:
            self.log_significant_event("About to teardown")
            self.cluster.teardown()
            self.log_significant_event("teardown completed")
        except Exception as err:
            self.log_significant_event("teardown failed")
            if hold is None:
                hold = err

        if hold is not None:
            raise hold

    def log_significant_event(self, msg):
        now = datetime.now()
        time = now.strftime("%H:%M:%S")
        marker = "****"
        print(marker)
        print(f"{marker} {time}: {msg}")
        print(marker)
