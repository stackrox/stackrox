#!/usr/bin/env python3

"""
Common test run patterns
"""

from clusters import NullCluster
from ci_tests import NullTest
from posts import NullPost


class ClusterTestRunner:
    def __init__(self, cluster=NullCluster(), test=NullTest(), post=NullPost()):
        self.cluster = cluster
        self.test = test
        self.post = post

    def run(self):
        hold = None
        try:
            self.cluster.provision()
        except Exception as err:
            hold = err
        if hold is None:
            try:
                self.test.run()
            except Exception as err:
                hold = err
            try:
                self.post.run(test_output_dirs=self.test.test_output_dirs)
            except Exception as err:
                if hold is None:
                    hold = err

        try:
            self.cluster.teardown()
        except Exception as err:
            if hold is None:
                hold = err

        if hold is not None:
            raise hold
