#!/usr/bin/env python3

"""
Unit tests for get_compatibility_test_tuples.py

These tests verify that version comparison works correctly,
particularly for double-digit minor versions (4.10+) where
string comparison fails.
"""

import unittest
import sys
from pathlib import Path

# Add parent directory to path to import the module under test
sys.path.insert(0, str(Path(__file__).parent.parent))

from get_compatibility_test_tuples import is_newer_version


class TestIsNewerVersion(unittest.TestCase):
    """Test the is_newer_version function to prevent regression of string vs int comparison bug"""

    def test_double_digit_minor_version_not_newer(self):
        """4.11 should NOT consider 4.9 as newer (this was the bug)"""
        self.assertFalse(
            is_newer_version('4.11.x-94-g75a2cb6b34', '40009.5.3'),
            "4.9 should not be considered newer than 4.11"
        )

    def test_double_digit_minor_version_4_10_vs_4_9(self):
        """4.10 should NOT consider 4.9 as newer"""
        self.assertFalse(
            is_newer_version('4.10.x-50-ghash', '40009.5.3'),
            "4.9 should not be considered newer than 4.10"
        )

    def test_double_digit_minor_version_4_11_vs_4_10(self):
        """4.11 should NOT consider 4.10 as newer"""
        self.assertFalse(
            is_newer_version('4.11.x-94-ghash', '40010.2.1'),
            "4.10 should not be considered newer than 4.11"
        )

    def test_double_digit_minor_version_is_newer(self):
        """4.10 should consider 4.11 as newer"""
        self.assertTrue(
            is_newer_version('4.10.x-50-ghash', '40011.1.0'),
            "4.11 should be considered newer than 4.10"
        )

    def test_single_digit_minor_version_not_newer(self):
        """4.8 should NOT consider 4.7 as newer (existing behavior)"""
        self.assertFalse(
            is_newer_version('4.8.x-325-ghash', '40007.0.0'),
            "4.7 should not be considered newer than 4.8"
        )

    def test_single_digit_minor_version_is_newer(self):
        """4.6 should consider 4.9 as newer (existing behavior)"""
        self.assertTrue(
            is_newer_version('4.6.x-736-ghash', '40009.5.3'),
            "4.9 should be considered newer than 4.6"
        )

    def test_equal_major_minor_versions(self):
        """Equal major.minor versions should not be considered newer"""
        self.assertFalse(
            is_newer_version('4.9.x-100-ghash', '40009.0.0'),
            "Same version should not be considered newer"
        )

    def test_major_version_difference(self):
        """Higher major version should be considered newer"""
        self.assertTrue(
            is_newer_version('4.11.x-94-ghash', '50000.0.0'),
            "5.0 should be considered newer than 4.11"
        )

    def test_major_version_not_newer(self):
        """Lower major version should not be considered newer"""
        self.assertFalse(
            is_newer_version('4.11.x-94-ghash', '30074.9.0'),
            "3.74 should not be considered newer than 4.11"
        )

    def test_release_candidate_version(self):
        """Test with release candidate versions"""
        self.assertTrue(
            is_newer_version('4.11.1-rc.2', '40012.0.0'),
            "4.12 should be considered newer than 4.11.1-rc.2"
        )

    def test_release_candidate_not_newer(self):
        """Test with release candidate versions - not newer"""
        self.assertFalse(
            is_newer_version('4.11.1-rc.2', '40010.5.0'),
            "4.10 should not be considered newer than 4.11.1-rc.2"
        )

    def test_patch_version_differences(self):
        """Test that patch versions are handled correctly"""
        # 4.11.2 should not consider 4.11.1 as newer
        self.assertFalse(
            is_newer_version('4.11.2-10-ghash', '40011.1.0'),
            "4.11.1 should not be considered newer than 4.11.2"
        )

    def test_three_digit_minor_version(self):
        """Test edge case with hypothetical three-digit minor versions"""
        # If we ever reach 4.100, this should work
        self.assertFalse(
            is_newer_version('4.100.x-1-ghash', '40099.0.0'),
            "4.99 should not be considered newer than 4.100"
        )

    def test_helm_version_with_different_patch_levels(self):
        """Test comparison with different helm chart patch versions"""
        self.assertFalse(
            is_newer_version('4.9.x-100-ghash', '40009.10.5'),
            "4.9.10 should not be considered newer than 4.9"
        )


if __name__ == '__main__':
    unittest.main()
