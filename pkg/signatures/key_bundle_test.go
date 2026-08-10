package signatures

import (
	"os"
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

func TestBundleJSONIsValid(t *testing.T) {
	data, err := os.ReadFile("bundle.json")
	require.NoError(t, err, "bundle.json must exist in pkg/signatures/")

	bundle, err := ParseKeyBundle(data)
	require.NoError(t, err, "bundle.json must be valid")
	assert.NotEmpty(t, bundle.CosignKeys, "bundle.json must contain at least one cosign key")
}

func TestBundleToSignatureIntegration(t *testing.T) {
	data, err := os.ReadFile("bundle.json")
	require.NoError(t, err)

	bundle, err := ParseKeyBundle(data)
	require.NoError(t, err)

	si, err := bundle.ToSignatureIntegration()
	require.NoError(t, err)
	assert.Equal(t, DefaultRedHatIntegrationID, si.GetId())
	assert.Equal(t, DefaultRedHatIntegrationName, si.GetName())
	assert.Equal(t, storage.Traits_DEFAULT, si.GetTraits().GetOrigin())

	keys := si.GetCosign().GetPublicKeys()
	assert.NotEmpty(t, keys, "integration must have at least one cosign key")
	for _, key := range keys {
		assert.NotEmpty(t, key.GetName(), "key name must not be empty")
		assert.NotEmpty(t, key.GetPublicKeyPemEnc(), "key PEM must not be empty")
	}
}

func TestParseKeyBundle(t *testing.T) {
	cases := map[string]struct {
		input   string
		wantErr error
	}{
		"valid single key": {
			input: `{"schemaVersion": "1.0", "cosignKeys": [{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"}]}`,
		},
		"valid multiple keys": {
			input: `{"schemaVersion": "1.0", "cosignKeys": [
				{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"},
				{"name": "key-2", "publicKey": "` + testKeyPEMJSON2 + `"}
			]}`,
		},
		"no key groups": {
			input:   `{"schemaVersion": "1.0"}`,
			wantErr: ErrKeyBundleEmpty,
		},
		"empty cosign keys array": {
			input:   `{"schemaVersion": "1.0", "cosignKeys": []}`,
			wantErr: ErrKeyBundleEmpty,
		},
		"empty object": {
			input:   `{}`,
			wantErr: ErrKeyBundleEmpty,
		},
		"only unknown groups": {
			input:   `{"schemaVersion": "1.0", "pgpKeys": [{"name": "k", "armoredKey": "opaque"}]}`,
			wantErr: ErrNoSupportedKeys,
		},
		"empty name": {
			input:   `{"cosignKeys": [{"name": "", "publicKey": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNameEmpty,
		},
		"whitespace-only name": {
			input:   `{"cosignKeys": [{"name": "  \t ", "publicKey": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNameEmpty,
		},
		"name with forward slash": {
			input:   `{"cosignKeys": [{"name": "foo/bar", "publicKey": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNamePathSeparator,
		},
		"name with backslash": {
			input:   `{"cosignKeys": [{"name": "foo\\bar", "publicKey": "` + testKeyPEMJSON + `"}]}`,
			wantErr: ErrKeyNamePathSeparator,
		},
		"invalid PEM": {
			input:   `{"cosignKeys": [{"name": "bad-key", "publicKey": "not-a-pem"}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"whitespace-only PEM": {
			input:   `{"cosignKeys": [{"name": "bad-key", "publicKey": "   \t\n  "}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"wrong PEM type": { //nolint:gosec // G101: test data, not real credentials
			input:   `{"cosignKeys": [{"name": "bad-key", "publicKey": "-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJB\n-----END RSA PRIVATE KEY-----\n"}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"valid + invalid key rejects entire bundle": {
			input: `{"cosignKeys": [
				{"name": "good", "publicKey": "` + testKeyPEMJSON + `"},
				{"name": "bad", "publicKey": "not-a-pem"}
			]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"trailing PEM data": {
			input:   `{"cosignKeys": [{"name": "key-1", "publicKey": "` + jsonEscapePEM(testPublicKeyPEM+"extra") + `"}]}`,
			wantErr: ErrKeyInvalidPEM,
		},
		"duplicate key names": {
			input: `{"cosignKeys": [
				{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"},
				{"name": "key-1", "publicKey": "` + testKeyPEMJSON2 + `"}
			]}`,
			wantErr: ErrKeyNameDuplicate,
		},
		"unknown schema version accepted": {
			input: `{"schemaVersion": "2.0", "cosignKeys": [{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"}]}`,
		},
		"unknown groups with cosign keys accepted": {
			input: `{"schemaVersion": "1.0", "cosignKeys": [{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"}], "pgpKeys": [{"name": "k", "armoredKey": "opaque"}]}`,
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

func TestParseKeyBundlePreservesFields(t *testing.T) {
	cases := map[string]struct {
		input             string
		wantSchemaVersion string
		wantKeyCount      int
	}{
		"v1.0 with cosign keys": {
			input:             `{"schemaVersion": "1.0", "cosignKeys": [{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"}]}`,
			wantSchemaVersion: SchemaVersion1,
			wantKeyCount:      1,
		},
		"multiple cosign keys": {
			input: `{"schemaVersion": "1.0", "cosignKeys": [
				{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"},
				{"name": "key-2", "publicKey": "` + testKeyPEMJSON2 + `"}
			]}`,
			wantSchemaVersion: SchemaVersion1,
			wantKeyCount:      2,
		},
		"missing version stays empty": {
			input:             `{"cosignKeys": [{"name": "key-1", "publicKey": "` + testKeyPEMJSON + `"}]}`,
			wantSchemaVersion: "",
			wantKeyCount:      1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			bundle, err := ParseKeyBundle([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.wantSchemaVersion, bundle.SchemaVersion)
			assert.Len(t, bundle.CosignKeys, tc.wantKeyCount)
		})
	}
}

func TestBundleToSignatureIntegrationAllCosignKeys(t *testing.T) {
	bundle := &KeyBundle{
		SchemaVersion: SchemaVersion1,
		CosignKeys: []CosignKey{
			{Name: "key-1", PublicKey: testPublicKeyPEM},
			{Name: "key-2", PublicKey: testPublicKeyPEM2},
		},
	}

	si, err := bundle.ToSignatureIntegration()
	require.NoError(t, err)
	keys := si.GetCosign().GetPublicKeys()
	require.Len(t, keys, 2)
	assert.Equal(t, "key-1", keys[0].GetName())
	assert.Equal(t, "key-2", keys[1].GetName())
}

func TestBundleToSignatureIntegrationRejectsNoCosignKeys(t *testing.T) {
	bundle := &KeyBundle{SchemaVersion: SchemaVersion1}

	si, err := bundle.ToSignatureIntegration()
	assert.ErrorIs(t, err, ErrNoSupportedKeys)
	assert.Nil(t, si)
}

func TestParseKeyBundleMalformedJSON(t *testing.T) {
	bundle, err := ParseKeyBundle([]byte(`{not json`))
	assert.ErrorIs(t, err, ErrUnmarshalling)
	assert.Nil(t, bundle)
}

func TestParseKeyBundlePEMCanonicalization(t *testing.T) {
	pemWithExtraNewlines := testPublicKeyPEM + "\n\n\n"
	input := `{"cosignKeys": [{"name": "key-1", "publicKey": "` + jsonEscapePEM(pemWithExtraNewlines) + `"}]}`

	bundle, err := ParseKeyBundle([]byte(input))
	require.NoError(t, err)
	require.Len(t, bundle.CosignKeys, 1)

	assert.Regexp(t, `\n$`, bundle.CosignKeys[0].PublicKey)
	assert.NotRegexp(t, `\n\n$`, bundle.CosignKeys[0].PublicKey)
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
