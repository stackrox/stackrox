package signatures

import (
	"encoding/base64"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stretchr/testify/assert"
)

func TestNewPublicKeyVerifier(t *testing.T) {
	cases := map[string]struct {
		base64EncKey string
		fail         bool
		err          error
	}{
		"valid public key": {
			base64EncKey: "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFMDRzb0Fv" +
				"TnlnUmhheXRDdHlnUGN3c1ArNkVpbgpZb0R2L0JKeDFUOVdtdHNBTmgySHBsUlI2NkZibSszT2pGdWFoMkloRnVmUGhEbDZhODVJM3l" +
				"tVll3PT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg==",
		},
		"error in decoding base64 encoded string": {
			base64EncKey: "<",
			fail:         true,
			err:          base64.CorruptInputError(0),
		},
		"non PEM encoded public key": {
			base64EncKey: "anVzdHNvbWV0ZXh0Cg==",
			fail:         true,
			err:          errorhelpers.ErrInvariantViolation,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			config := &storage.SignatureVerificationConfig_PublicKey{PublicKey: &storage.CosignPublicKeyVerification{PublicKeysBase64Enc: []string{c.base64EncKey}}}
			verifier, err := newPublicKeyVerifier(config)
			if c.fail {
				assert.Error(t, err)
				assert.Nil(t, verifier)
				assert.ErrorIs(t, err, c.err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, verifier.parsedPublicKeys, 1)
			}
		})
	}
}
