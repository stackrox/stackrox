package signatures

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
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
			wantErr: ErrKeyBundleEmpty,
		},
		"missing keys field": {
			input:   `{}`,
			wantErr: ErrKeyBundleEmpty,
		},
		"empty name": {
			input:   `{"keys": [{"name": "", "pem": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNameEmpty,
		},
		"whitespace-only name": {
			input:   `{"keys": [{"name": "  \t ", "pem": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNameEmpty,
		},
		"name with forward slash": {
			input:   `{"keys": [{"name": "foo/bar", "pem": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNamePathSeparator,
		},
		"name with backslash": {
			input:   `{"keys": [{"name": "foo\\bar", "pem": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNamePathSeparator,
		},
		"invalid PEM": {
			input:   `{"keys": [{"name": "bad-key", "pem": "not-a-pem"}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"whitespace-only PEM": {
			input:   `{"keys": [{"name": "bad-key", "pem": "   \t\n  "}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"wrong PEM type": { //nolint:gosec // G101: test data, not real credentials
			input:   `{"keys": [{"name": "bad-key", "pem": "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJB\n-----END RSA PRIVATE KEY-----\n"}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"valid + invalid key rejects entire bundle": {
			input: `{"keys": [
				{"name": "good", "pem": "` + testKeyPEMJSON + `"},
				{"name": "bad", "pem": "not-a-pem"}
			]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"trailing PEM data": {
			input:   `{"keys": [{"name": "key-1", "pem": "` + jsonEscapePEM(testPublicKeyPEM+"extra") + `"}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"duplicate key names": {
			input: `{"keys": [
				{"name": "key-1", "pem": "` + testKeyPEMJSON + `"},
				{"name": "key-1", "pem": "` + testKeyPEMJSON2 + `"}
			]}`,
			wantErr: ErrKeyNameDuplicate,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			bundle, err := ParseKeyBundle([]byte(tc.input))
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
	bundle, err := ParseKeyBundle([]byte(`{not json`))
	assert.ErrorContains(t, err, "unmarshalling key bundle JSON")
	assert.Nil(t, bundle)
}

func TestParseKeyBundlePEMCanonicalization(t *testing.T) {
	pemWithExtraNewlines := testPublicKeyPEM + "\n\n\n"
	input := `{"keys": [{"name": "key-1", "pem": "` + jsonEscapePEM(pemWithExtraNewlines) + `"}]}`

	bundle, err := ParseKeyBundle([]byte(input))
	require.NoError(t, err)
	require.Len(t, bundle.Keys, 1)

	assert.Regexp(t, `\n$`, bundle.Keys[0].PEM)
	assert.NotRegexp(t, `\n\n$`, bundle.Keys[0].PEM)
}

func TestBundleToSignatureIntegration(t *testing.T) {
	bundle := &KeyBundle{
		Keys: []KeyBundleEntry{
			{Name: "key-1", PEM: testPublicKeyPEM},
			{Name: "key-2", PEM: testPublicKeyPEM2},
		},
	}

	si := BundleToSignatureIntegration(bundle)

	assert.Equal(t, DefaultRedHatSignatureIntegration.GetId(), si.GetId())
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
