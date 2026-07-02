package cryptocodec

import (
	"encoding/base64"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestGCMBackwardCompatibility(t *testing.T) {
	// nonce(12) || ciphertext || tag(16) produced by the old cipher.NewGCM + manual nonce prepend.
	// NewGCMWithRandomNonce uses the same wire format so must decrypt this.
	keyString := "QUVTMjU2S2V5LTMyQ2hhcmFjdGVyczEyMzQ1Njc4OTA="
	oldCiphertext := "HnI7658TcuhmqkfeG2ufIo/pIq3fimbZtPGUgSUpoF99IySLdhDi"

	codec := NewGCMCryptoCodec()
	decrypted, err := codec.Decrypt(keyString, oldCiphertext)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", decrypted)
}

func TestGCMEncryptionDecryption(t *testing.T) {
	// Test string encryption/decryption
	originalText := "lorem ipsum dolor sit amet"
	keyString := base64.StdEncoding.EncodeToString([]byte("AES256Key-32Characters1234567890"))
	codec := NewGCMCryptoCodec()

	cryptoText, err := codec.Encrypt(keyString, originalText)
	assert.NoError(t, err)

	decryptedText, err := codec.Decrypt(keyString, cryptoText)
	assert.NoError(t, err)
	assert.Equal(t, originalText, decryptedText)

	// Test struct encryption/decryption
	// #nosec G101 -- test data
	originalCreds := &storage.AWSSecurityHub_Credentials{
		AccessKeyId:     "key-id",
		SecretAccessKey: "lorem ipsum dolor sit amet",
	}
	marshalled, err := originalCreds.MarshalVT()
	assert.NoError(t, err)
	marshalledString := string(marshalled)

	cryptoText, err = codec.Encrypt(keyString, marshalledString)
	assert.NoError(t, err)

	decryptedText, err = codec.Decrypt(keyString, cryptoText)
	assert.NoError(t, err)
	decryptedBytes := []byte(decryptedText)
	decryptedCreds := &storage.AWSSecurityHub_Credentials{}
	err = decryptedCreds.UnmarshalVTUnsafe(decryptedBytes)
	assert.NoError(t, err)
	protoassert.Equal(t, originalCreds, decryptedCreds)
}
