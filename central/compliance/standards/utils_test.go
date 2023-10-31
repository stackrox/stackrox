package standards

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSupportedStandards(t *testing.T) {
	expectedStandards := []string{
		"HIPAA_164",
		"NIST_800_190",
		"NIST_SP_800_53_Rev_4",
		"PCI_DSS_3_2",
		"CIS_Kubernetes_v1_5",
	}
	assert.ElementsMatch(t, expectedStandards, GetSupportedStandards())
}

func TestFilterSupported(t *testing.T) {
	standards := []string{
		"CIS_Docker_v1_1_0",
		"CIS_Docker_v1_2_0",
		"CIS_Kubernetes_v1_5",
	}

	expectedStandards := []string{
		"CIS_Kubernetes_v1_5",
	}

	supportedStandards, _ := FilterSupported(standards)
	assert.ElementsMatch(t, supportedStandards, expectedStandards)
}
