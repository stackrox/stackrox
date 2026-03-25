import os
import unittest

from common import enable_sfa_for_ocp


class TestEnableSfaForOcp(unittest.TestCase):
    def setUp(self):
        os.environ.pop("SFA_AGENT", None)
        os.environ.pop("CLUSTER_FLAVOR_VARIANT", None)

    def tearDown(self):
        os.environ.pop("SFA_AGENT", None)
        os.environ.pop("CLUSTER_FLAVOR_VARIANT", None)

    def test_enables_sfa_for_ocp_4_16(self):
        os.environ["CLUSTER_FLAVOR_VARIANT"] = "openshift-4-ocp/stable-4.16"
        enable_sfa_for_ocp()
        self.assertEqual(os.environ.get("SFA_AGENT"), "true")

    def test_enables_sfa_for_ocp_4_17(self):
        os.environ["CLUSTER_FLAVOR_VARIANT"] = "openshift-4-ocp/stable-4.17"
        enable_sfa_for_ocp()
        self.assertEqual(os.environ.get("SFA_AGENT"), "true")

    def test_does_not_enable_sfa_for_ocp_4_15(self):
        os.environ["CLUSTER_FLAVOR_VARIANT"] = "openshift-4-ocp/stable-4.15"
        enable_sfa_for_ocp()
        self.assertIsNone(os.environ.get("SFA_AGENT"))

    def test_does_not_enable_sfa_when_variant_missing(self):
        enable_sfa_for_ocp()
        self.assertIsNone(os.environ.get("SFA_AGENT"))

    def test_does_not_enable_sfa_for_non_ocp_variant(self):
        os.environ["CLUSTER_FLAVOR_VARIANT"] = "gke-default"
        enable_sfa_for_ocp()
        self.assertIsNone(os.environ.get("SFA_AGENT"))

    def test_does_not_enable_sfa_for_malformed_variant(self):
        os.environ["CLUSTER_FLAVOR_VARIANT"] = "openshift-4-ocp/"
        enable_sfa_for_ocp()
        self.assertIsNone(os.environ.get("SFA_AGENT"))


if __name__ == "__main__":
    unittest.main()
