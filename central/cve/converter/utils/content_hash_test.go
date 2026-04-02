package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestComputeCVEContentHash_Deterministic(t *testing.T) {
	// Same inputs should produce same hash.
	hash1 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	hash2 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	assert.Equal(t, hash1, hash2, "Same inputs should produce identical hashes")
}

func TestComputeCVEContentHash_SeverityDifference(t *testing.T) {
	// Inputs differing only in severity should produce different hashes.
	hash1 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	hash2 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "HIGH", 7.5, "Test summary")
	assert.NotEqual(t, hash1, hash2, "Different severity should produce different hashes")
}

func TestComputeCVEContentHash_CVSSV3Difference(t *testing.T) {
	// Inputs differing only in cvss_v3 should produce different hashes.
	hash1 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	hash2 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 8.0, "Test summary")
	assert.NotEqual(t, hash1, hash2, "Different CVSS v3 score should produce different hashes")
}

func TestComputeCVEContentHash_ZeroCVSS(t *testing.T) {
	// cvss_v3 = 0.0 should use empty string, not "0.00".
	hashZero := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 0.0, "Test summary")
	hashSmall := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 0.01, "Test summary")
	assert.NotEqual(t, hashZero, hashSmall, "CVSS 0.0 should use empty string, different from 0.01")

	// Verify zero CVSS produces a hash (not empty).
	assert.NotEmpty(t, hashZero, "Hash with zero CVSS should not be empty")
}

func TestComputeCVEContentHash_BoundaryCollisionPrevention(t *testing.T) {
	// Boundary collision attack: cveName with embedded null byte should differ from
	// cveName without null byte but different source.
	hash1 := ComputeCVEContentHash("CVE-2024-1234\x00NVD", "", "CRITICAL", 7.5, "Test summary")
	hash2 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	assert.NotEqual(t, hash1, hash2, "Embedded null byte should produce different hash")

	// Another boundary test: different field splits.
	hash3 := ComputeCVEContentHash("CVE-2024", "1234", "CRITICAL", 7.5, "Test summary")
	hash4 := ComputeCVEContentHash("CVE-20241234", "", "CRITICAL", 7.5, "Test summary")
	assert.NotEqual(t, hash3, hash4, "Different field boundaries should produce different hashes")
}

func TestComputeCVEContentHash_AdvisoryNotIncluded(t *testing.T) {
	// Changing advisory_name should NOT change the hash, since it's not a parameter.
	// This is implicitly tested by the function signature not having advisory_name.
	// We verify by checking that the hash depends only on the provided parameters.
	hash1 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	hash2 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	assert.Equal(t, hash1, hash2, "Hash should be consistent without advisory_name parameter")

	// Different summary should change hash (verifying summary is included).
	hash3 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Different summary")
	assert.NotEqual(t, hash1, hash3, "Different summary should produce different hash")
}

func TestComputeCVEContentHash_CVSSV3Formatting(t *testing.T) {
	// Verify CVSS v3 is formatted to 2 decimal places.
	hash1 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Test summary")
	hash2 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.50, "Test summary")
	assert.Equal(t, hash1, hash2, "7.5 and 7.50 should produce same hash")

	// Verify formatting to 2 decimal places with rounding.
	hash3 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.504, "Test summary")
	hash4 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.505, "Test summary")
	// 7.504 rounds down to 7.50, 7.505 rounds up to 7.51.
	assert.NotEqual(t, hash3, hash4, "7.504 and 7.505 should produce different hashes due to rounding")
}

func TestComputeCVEContentHash_EmptyFields(t *testing.T) {
	// Test with empty strings for various fields.
	hash1 := ComputeCVEContentHash("", "", "", 0.0, "")
	assert.NotEmpty(t, hash1, "Hash with all empty fields should not be empty")

	// Empty summary should differ from non-empty.
	hash2 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "")
	hash3 := ComputeCVEContentHash("CVE-2024-1234", "NVD", "CRITICAL", 7.5, "Summary")
	assert.NotEqual(t, hash2, hash3, "Empty summary should produce different hash")
}

func TestSourceToString(t *testing.T) {
	tests := []struct {
		source   storage.Source
		expected string
	}{
		{storage.Source_SOURCE_OSV, "OSV"},
		{storage.Source_SOURCE_NVD, "NVD"},
		{storage.Source_SOURCE_RED_HAT, "RED_HAT"},
		{storage.Source_SOURCE_UNKNOWN, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := SourceToString(tt.source)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverityToString(t *testing.T) {
	tests := []struct {
		severity storage.VulnerabilitySeverity
		expected string
	}{
		{storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, "CRITICAL"},
		{storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, "HIGH"},
		{storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, "MEDIUM"},
		{storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY, "LOW"},
		{storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := SeverityToString(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}
