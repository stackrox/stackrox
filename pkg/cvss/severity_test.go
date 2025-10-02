package cvss

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestVulnToSeverity(t *testing.T) {
	_ = t
	tests := map[string]struct {
		input *vulnScoreInfo
		want  storage.VulnerabilitySeverity
	}{
		"known severity is propagated": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			},
			want: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"unknown severity is returned if no alternative source exists": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
			},
			want: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
		"unknown severity is returned if no source has known severity": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
				cvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_UNKNOWN,
				},
				cvssv2: &storage.CVSSV2{
					Severity: storage.CVSSV2_UNKNOWN,
				},
			},
			want: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
		"CVSS V2 severity is used if the base severity is unknown and V3 is not available": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
				cvssv2: &storage.CVSSV2{
					Severity: storage.CVSSV2_LOW,
				},
			},
			want: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"unknown CVSS V3 takes precedence over CVSS V2 severity if the base severity is unknown": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
				cvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_UNKNOWN,
				},
				cvssv2: &storage.CVSSV2{
					Severity: storage.CVSSV2_LOW,
				},
			},
			want: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
		"known CVSS V3 severity is used if the base severity is unknown": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
				cvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
			},
			want: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		"known CVSS V3 takes precedence over CVSS V2 severity if the base severity is unknown": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
				cvssv2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
				cvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_LOW,
				},
			},
			want: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"known base severity takes precedence over provided CVSSV3 one": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				cvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
			},
			want: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"known base severity takes precedence over provided CVSSV2 one": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				cvssv2: &storage.CVSSV2{
					Severity: storage.CVSSV2_MEDIUM,
				},
			},
			want: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"known base severity takes precedence over provided CVSSV2 and CVSSV3 ones": {
			input: &vulnScoreInfo{
				severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				cvssV3: &storage.CVSSV3{
					Severity: storage.CVSSV3_MEDIUM,
				},
				cvssv2: &storage.CVSSV2{
					Severity: storage.CVSSV2_HIGH,
				},
			},
			want: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(it *testing.T) {
			result := VulnToSeverity(tc.input)
			assert.Equal(it, tc.want, result)
		})
	}
}

func TestConvertCVSSV2SeverityToVulnerabilitySeverity(t *testing.T) {
	tests := map[string]struct {
		input storage.CVSSV2_Severity
		want  storage.VulnerabilitySeverity
	}{
		"UNKNOWN": {
			input: storage.CVSSV2_UNKNOWN,
			want:  storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
		"LOW": {
			input: storage.CVSSV2_LOW,
			want:  storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"MEDIUM": {
			input: storage.CVSSV2_MEDIUM,
			want:  storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		"HIGH": {
			input: storage.CVSSV2_HIGH,
			want:  storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		},
		"Default": {
			input: storage.CVSSV2_Severity(-1),
			want:  storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(it *testing.T) {
			result := ConvertCVSSV2SeverityToVulnerabilitySeverity(tc.input)
			assert.Equal(it, tc.want, result)
		})
	}
}

func TestConvertCVSSV3SeverityToVulnerabilitySeverity(t *testing.T) {
	tests := map[string]struct {
		input storage.CVSSV3_Severity
		want  storage.VulnerabilitySeverity
	}{
		"UNKNOWN": {
			input: storage.CVSSV3_UNKNOWN,
			want:  storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
		"NONE": {
			input: storage.CVSSV3_NONE,
			want:  storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"LOW": {
			input: storage.CVSSV3_LOW,
			want:  storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		},
		"MEDIUM": {
			input: storage.CVSSV3_MEDIUM,
			want:  storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		},
		"HIGH": {
			input: storage.CVSSV3_HIGH,
			want:  storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		},
		"CRITICAL": {
			input: storage.CVSSV3_CRITICAL,
			want:  storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		},
		"Default": {
			input: storage.CVSSV3_Severity(-1),
			want:  storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(it *testing.T) {
			result := ConvertCVSSV3SeverityToVulnerabilitySeverity(tc.input)
			assert.Equal(it, tc.want, result)
		})
	}
}
