package signatures

import (
	_ "embed"

	"github.com/stackrox/rox/generated/storage"
)

const (
	// SignatureIntegrationIDPrefix should be prepended to every human-hostile ID of a
	// signature integration for readability, e.g.,
	//
	//	"io.stackrox.signatureintegration.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	SignatureIntegrationIDPrefix = "io.stackrox.signatureintegration."
)

// TODO Implement dynamic fetching to handle key rotations: ROX-29936
//
//go:embed "release-key-3.pub.txt"
var releaseKey3PublicKey string

var DefaultRedHatSignatureIntegration = &storage.SignatureIntegration{
	// Please don't change this ID, as it's referred to from other places. A migration may be needed if this is changed.
	Id:   SignatureIntegrationIDPrefix + "12a37a37-760e-4388-9e79-d62726c075b2",
	Name: "Red Hat",
	Cosign: &storage.CosignPublicKeyVerification{
		PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
			{
				Name:            "Red Hat Release Key 3",
				PublicKeyPemEnc: releaseKey3PublicKey,
			},
		},
	},
	CosignCertificates: nil,
	TransparencyLog: &storage.TransparencyLogVerification{
		Enabled:         false,
		Url:             "",
		ValidateOffline: false,
		PublicKeyPemEnc: "",
	},
}
