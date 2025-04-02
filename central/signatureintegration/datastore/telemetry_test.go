package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGather(t *testing.T) {

	assert.Empty(t, computeTelemetryProperties(context.Background(), []*storage.SignatureIntegration{}))

	props := computeTelemetryProperties(context.Background(), []*storage.SignatureIntegration{
		{},
		{
			Cosign: goodCosignConfig,
		},
		{
			CosignCertificates: goodCosignCertificateVerificationConfig,
		},
		{
			Cosign:             goodCosignConfig,
			CosignCertificates: goodCosignCertificateVerificationConfig,
		},
		{
			Cosign: goodCosignConfig,
			CosignCertificates: []*storage.CosignCertificateVerification{
				{
					CertificatePemEnc: "===",
				},
			},
		},
	})

	expectedProps := map[string]any{
		"Total Signature Integration Certificates":            3,
		"Total Signature Integration Cosign Public Keys":      3,
		"Total Signature Integration With Custom Certificate": 1,
		"Total Signature Integration With Custom Chain":       2,
		"Total Signature Integrations":                        5,
	}
	assert.Equal(t, expectedProps, props)
}
