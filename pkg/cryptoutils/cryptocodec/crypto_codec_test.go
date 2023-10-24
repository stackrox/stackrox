package cryptocodec

import (
	"encoding/base64"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

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
	originalCreds := &storage.AWSSecurityHub_Credentials{
		AccessKeyId:     "key-id",
		SecretAccessKey: "lorem ipsum dolor sit amet",
	}
	marshalled, err := originalCreds.Marshal()
	assert.NoError(t, err)
	marshalledString := string(marshalled)

	cryptoText, err = codec.Encrypt(keyString, marshalledString)
	assert.NoError(t, err)

	decryptedText, err = codec.Decrypt(keyString, cryptoText)
	assert.NoError(t, err)
	decryptedBytes := []byte(decryptedText)
	decryptedCreds := &storage.AWSSecurityHub_Credentials{}
	err = decryptedCreds.Unmarshal(decryptedBytes)
	assert.NoError(t, err)
	assert.Equal(t, originalCreds, decryptedCreds)
}
