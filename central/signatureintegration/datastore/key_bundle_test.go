package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
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

var (
	testKeyPEMJSON  = jsonEscapePEM(testPublicKeyPEM)
	testKeyPEMJSON2 = jsonEscapePEM(testPublicKeyPEM2)
)

func TestParseKeyBundle(t *testing.T) {
	cases := map[string]struct {
		input   string
		wantErr error
	}{
		"valid single key": {
			input: `{"keys": [{"name": "key-1", "pem": "` + testKeyPEMJSON + `"}]}`,
		},
		"valid multiple keys": {
			input: `{"keys": [
				{"name": "key-1", "pem": "` + testKeyPEMJSON + `"},
				{"name": "key-2", "pem": "` + testKeyPEMJSON2 + `"}
			]}`,
		},
		"empty keys array": {
			input:   `{"keys": []}`,
			wantErr: errKeyBundleEmpty,
		},
		"missing keys field": {
			input:   `{}`,
			wantErr: errKeyBundleEmpty,
		},
		"empty name": {
			input:   `{"keys": [{"name": "", "pem": "` + testKeyPEMJSON + `"}]}`,
			wantErr: errKeyNameEmpty,
		},
		"whitespace-only name": {
			input:   `{"keys": [{"name": "  \t ", "pem": "` + testKeyPEMJSON + `"}]}`,
			wantErr: errKeyNameEmpty,
		},
		"name with path separator": {
			input:   `{"keys": [{"name": "foo/bar", "pem": "` + testKeyPEMJSON + `"}]}`,
			wantErr: errKeyNamePathSeparator,
		},
		"invalid PEM": {
			input:   `{"keys": [{"name": "bad-key", "pem": "not-a-pem"}]}`,
			wantErr: errKeyInvalidPEM,
		},
		"wrong PEM type": { //nolint:gosec // G101: test data, not real credentials
			input:   `{"keys": [{"name": "bad-key", "pem": "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJB\n-----END RSA PRIVATE KEY-----\n"}]}`,
			wantErr: errKeyInvalidPEM,
		},
		"valid + invalid key rejects entire bundle": {
			input: `{"keys": [
				{"name": "good", "pem": "` + testKeyPEMJSON + `"},
				{"name": "bad", "pem": "not-a-pem"}
			]}`,
			wantErr: errKeyInvalidPEM,
		},
		"trailing PEM data": {
			input:   `{"keys": [{"name": "key-1", "pem": "` + jsonEscapePEM(testPublicKeyPEM+"extra") + `"}]}`,
			wantErr: errKeyInvalidPEM,
		},
		"duplicate key names": {
			input: `{"keys": [
				{"name": "key-1", "pem": "` + testKeyPEMJSON + `"},
				{"name": "key-1", "pem": "` + testKeyPEMJSON2 + `"}
			]}`,
			wantErr: errKeyNameDuplicate,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			bundle, err := parseKeyBundle([]byte(tc.input))
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, bundle)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, bundle)
			}
		})
	}
}

func TestParseKeyBundleMalformedJSON(t *testing.T) {
	bundle, err := parseKeyBundle([]byte(`{not json`))
	assert.ErrorContains(t, err, "unmarshalling key bundle JSON")
	assert.Nil(t, bundle)
}

func TestParseKeyBundlePEMCanonicalization(t *testing.T) {
	pemWithExtraNewlines := testPublicKeyPEM + "\n\n\n"
	input := `{"keys": [{"name": "key-1", "pem": "` + jsonEscapePEM(pemWithExtraNewlines) + `"}]}`

	bundle, err := parseKeyBundle([]byte(input))
	require.NoError(t, err)
	require.Len(t, bundle.Keys, 1)

	assert.Regexp(t, `\n$`, bundle.Keys[0].PEM)
	assert.NotRegexp(t, `\n\n$`, bundle.Keys[0].PEM)
}

func TestToDefaultSignatureIntegration(t *testing.T) {
	bundle := &keyBundle{
		Keys: []keyBundleEntry{
			{Name: "key-1", PEM: testPublicKeyPEM},
			{Name: "key-2", PEM: testPublicKeyPEM2},
		},
	}

	si := bundle.toDefaultSignatureIntegration()

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
