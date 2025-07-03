package signatures

import (
	"github.com/stackrox/rox/generated/storage"
)

const (
	// SignatureIntegrationIDPrefix should be prepended to every human-hostile ID of a
	// signature integration for readability, e.g.,
	//
	//	"io.stackrox.signatureintegration.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	SignatureIntegrationIDPrefix = "io.stackrox.signatureintegration."

	// TODO Implement dynamic fetching to handle key rotations: ROX-29936
	releaseKey3PublicKey = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA0ASyuH2TLWvBUqPHZ4Ip
75g7EncBkgQHdJnjzxAW5KQTMh/siBoB/BoSrtiPMwnChbTCnQOIQeZuDiFnhuJ7
M/D3b7JoX0m123NcCSn67mAdjBa6Bg6kukZgCP4ZUZeESajWX/EjylFcRFOXW57p
RDCEN42J/jYlVqt+g9+Grker8Sz86H3l0tbqOdjbz/VxHYhwF0ctUMHsyVRDq2QP
tqzNXlmlMhS/PoFr6R4u/7HCn/K+LegcO2fAFOb40KvKSKKVD6lewUZErhop1CgJ
XjDtGmmO9dGMF71mf6HEfaKSdy+EE6iSF2A2Vv9QhBawMiq2kOzEiLg4nAdJT8wg
ZrMAmPCqGIsXNGZ4/Q+YTwwlce3glqb5L9tfNozEdSR9N85DESfQLQEdY3CalwKM
BT1OEhEX1wHRCU4drMOej6BNW0VtscGtHmCrs74jPezhwNT8ypkyS+T0zT4Tsy6f
VXkJ8YSHyenSzMB2Op2bvsE3grY+s74WhG9UIA6DBxcTie15NSzKwfzaoNWODcLF
p7BY8aaHE2MqFxYFX+IbjpkQRfaeQQsouDFdCkXEFVfPpbD2dk6FleaMTPuyxtIT
gjVEtGQK2qGCFGiQHFd4hfV+eCA63Jro1z0zoBM5BbIIQ3+eVFwt3AlZp5UVwr6d
secqki/yrmv3Y0dqZ9VOn3UCAwEAAQ==
-----END PUBLIC KEY-----
`
)

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
