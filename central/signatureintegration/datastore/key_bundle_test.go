package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Test-only ECDSA P-256 keys, not real Red Hat keys.
	testPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE16IoQbiiB5exTRLTkl2rn5FuyXys
4TbDn4+GhQD1JmLZnAiA0cXktX+gFdxu/0JM9pcjjaqT7pdXztbBs78cXg==
-----END PUBLIC KEY-----
`
	testPublicKeyPEM2 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEQq1X/6XxCA4s0++8Tvl8k+Z0G/GN
LKpdYJEldXnyRE4ppY5d7vnRZHvdZQMSE3KoRSMvVnzZtc9LTKLB3DlS/w==
-----END PUBLIC KEY-----
`
)

func TestParseKeyBundle(t *testing.T) {
	cases := map[string]struct {
		input   string
		wantErr string
	}{
		"valid single key": {
			input: `{"keys": [{"name": "key-1", "pem": "` + jsonEscapePEM(testPublicKeyPEM) + `"}]}`,
		},
		"valid multiple keys": {
			input: `{"keys": [
				{"name": "key-1", "pem": "` + jsonEscapePEM(testPublicKeyPEM) + `"},
				{"name": "key-2", "pem": "` + jsonEscapePEM(testPublicKeyPEM2) + `"}
			]}`,
		},
		"empty keys array": {
			input:   `{"keys": []}`,
			wantErr: "at least one key",
		},
		"missing keys field": {
			input:   `{}`,
			wantErr: "at least one key",
		},
		"malformed JSON": {
			input:   `{not json`,
			wantErr: "unmarshalling key bundle JSON",
		},
		"empty name": {
			input:   `{"keys": [{"name": "", "pem": "` + jsonEscapePEM(testPublicKeyPEM) + `"}]}`,
			wantErr: "empty name",
		},
		"name with path separator": {
			input:   `{"keys": [{"name": "foo/bar", "pem": "` + jsonEscapePEM(testPublicKeyPEM) + `"}]}`,
			wantErr: "path separators",
		},
		"invalid PEM": {
			input:   `{"keys": [{"name": "bad-key", "pem": "not-a-pem"}]}`,
			wantErr: "invalid PEM-encoded public key",
		},
		"wrong PEM type": {
			input:   `{"keys": [{"name": "bad-key", "pem": "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJB\n-----END RSA PRIVATE KEY-----\n"}]}`, //nolint:gosec // G101: test data, not real credentials
			wantErr: "invalid PEM-encoded public key",
		},
		"valid + invalid key rejects entire bundle": {
			input: `{"keys": [
				{"name": "good", "pem": "` + jsonEscapePEM(testPublicKeyPEM) + `"},
				{"name": "bad", "pem": "not-a-pem"}
			]}`,
			wantErr: "invalid PEM-encoded public key",
		},
		"trailing PEM data": {
			input:   `{"keys": [{"name": "key-1", "pem": "` + jsonEscapePEM(testPublicKeyPEM+"extra") + `"}]}`,
			wantErr: "invalid PEM-encoded public key",
		},
		"duplicate key names": {
			input: `{"keys": [
				{"name": "key-1", "pem": "` + jsonEscapePEM(testPublicKeyPEM) + `"},
				{"name": "key-1", "pem": "` + jsonEscapePEM(testPublicKeyPEM2) + `"}
			]}`,
			wantErr: "duplicate key name",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			bundle, err := parseKeyBundle([]byte(tc.input))
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				assert.Nil(t, bundle)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, bundle)
			}
		})
	}
}

func TestParseKeyBundlePEMCanonicalization(t *testing.T) {
	// PEM with extra trailing newlines should be canonicalized.
	pemWithExtraNewlines := testPublicKeyPEM + "\n\n\n"
	input := `{"keys": [{"name": "key-1", "pem": "` + jsonEscapePEM(pemWithExtraNewlines) + `"}]}`

	bundle, err := parseKeyBundle([]byte(input))
	require.NoError(t, err)
	require.Len(t, bundle.Keys, 1)

	// Canonicalized PEM ends with exactly one trailing newline.
	assert.Regexp(t, `\n$`, bundle.Keys[0].PEM)
	assert.NotRegexp(t, `\n\n$`, bundle.Keys[0].PEM)
}

func TestToSignatureIntegration(t *testing.T) {
	bundle := &keyBundle{
		Keys: []keyBundleEntry{
			{Name: "key-1", PEM: testPublicKeyPEM},
			{Name: "key-2", PEM: testPublicKeyPEM2},
		},
	}

	si := bundle.toSignatureIntegration()

	assert.Equal(t, signatures.DefaultRedHatSignatureIntegration.GetId(), si.GetId())
	assert.Equal(t, "Red Hat", si.GetName())
	assert.Equal(t, storage.Traits_DEFAULT, si.GetTraits().GetOrigin())

	keys := si.GetCosign().GetPublicKeys()
	require.Len(t, keys, 2)
	assert.Equal(t, "key-1", keys[0].GetName())
	assert.Equal(t, testPublicKeyPEM, keys[0].GetPublicKeyPemEnc())
	assert.Equal(t, "key-2", keys[1].GetName())
	assert.Equal(t, testPublicKeyPEM2, keys[1].GetPublicKeyPemEnc())
}

func jsonEscapePEM(s string) string {
	var out []byte
	for _, c := range []byte(s) {
		switch c {
		case '\n':
			out = append(out, '\\', 'n')
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
