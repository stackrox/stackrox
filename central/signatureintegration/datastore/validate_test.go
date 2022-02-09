package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestValidateSignatureIntegration(t *testing.T) {
	goodName := "Sheer Heart Attack"
	goodID := "io.stackrox.signatureintegration.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13"
	badID := "Killer Queen"
	badEmptyCosignConfig := &storage.SignatureVerificationConfig_CosignVerification{
		CosignVerification: &storage.CosignPublicKeyVerification{},
	}
	badPEMEncodingCosignConfig := &storage.SignatureVerificationConfig_CosignVerification{
		CosignVerification: &storage.CosignPublicKeyVerification{
			PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
				{
					PublicKeyPemEnc: "this is not PEM encoded",
				},
			},
		},
	}
	goodCosignConfig := &storage.SignatureVerificationConfig_CosignVerification{
		CosignVerification: &storage.CosignPublicKeyVerification{
			PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
				{
					PublicKeyPemEnc: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO\n3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX\n7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS\nj+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd\nOrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ\n5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl\nFQIDAQAB\n-----END PUBLIC KEY-----",
				},
			},
		},
	}
	goodCosignConfigWithNamedKey := &storage.SignatureVerificationConfig_CosignVerification{
		CosignVerification: &storage.CosignPublicKeyVerification{
			PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
				{
					Name:            "Crazy Diamond",
					PublicKeyPemEnc: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO\n3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX\n7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS\nj+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd\nOrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ\n5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl\nFQIDAQAB\n-----END PUBLIC KEY-----",
				},
			},
		},
	}

	testCasesBad := map[string]*storage.SignatureIntegration{
		"name field must be set": {
			Id: goodID,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
				{
					Config: goodCosignConfig,
				},
			},
		},
		"id must follow format": {
			Id:   badID,
			Name: goodName,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
				{
					Config: goodCosignConfig,
				},
			},
		},
		"id field must be set": {
			Name: goodName,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
				{
					Config: goodCosignConfig,
				},
			},
		},
		"at least one signature verification config should be present": {
			Id:                           goodID,
			Name:                         goodName,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{},
		},
		"at least one public key in cosign config should be present": {
			Id:   goodID,
			Name: goodName,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
				{
					Config: badEmptyCosignConfig,
				},
			},
		},
		"public keys in cosign config should be PEM-encoded": {
			Id:   goodID,
			Name: goodName,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
				{
					Config: badPEMEncodingCosignConfig,
				},
			},
		},
	}

	testCasesGood := map[string]*storage.SignatureIntegration{
		"valid name, id and cosign config": {
			Id:   goodID,
			Name: goodName,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
				{
					Config: goodCosignConfig,
				},
			},
		},
		"named public keys in cosign config are valid": {
			Id:   goodID,
			Name: goodName,
			SignatureVerificationConfigs: []*storage.SignatureVerificationConfig{
				{
					Config: goodCosignConfigWithNamedKey,
				},
			},
		},
	}

	for desc, signatureIntegration := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSignatureIntegration(signatureIntegration)
			assert.NoErrorf(t, err, "signature integration: '%+v'", signatureIntegration)
		})
	}

	for desc, signatureIntegration := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSignatureIntegration(signatureIntegration)
			assert.Errorf(t, err, "signature integration: '%+v'", signatureIntegration)
		})
	}
}
