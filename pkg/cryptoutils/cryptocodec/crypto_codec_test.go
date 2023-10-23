package cryptocodec

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
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

func TestEncryption(t *testing.T) {
	s := "0a2436613332346462322d646338332d346264322d613961642d32656432373838353261306312187465737420696e746567726174696f6e2d636861726d696b1a0e6177735365637572697479487562221668747470733a2f2f6c6f63616c686f73743a383030308a01390a0975732d656173742d31121e0a0d71776572747931323334353637120d617364666768313233343536371a0c3031323334353637383931309a0150444e476775366339775570766b61712b6c78716671626e4c7334494967757452336a434d7863784f2f4b6e61715853325370363548385349586a756574316d596b6f2f51777a6d697937767152513d3d"
	data, err := hex.DecodeString(s)
	assert.NoError(t, err)
	notifier := &storage.Notifier{}
	err = notifier.Unmarshal(data)
	assert.NoError(t, err)
	fmt.Println(notifier.NotifierSecret)

	decrypted, err := Singleton().Decrypt("QUVTMjU2S2V5LTMyQ2hhcmFjdGVyczEyMzQ1Njc4OTA=", notifier.NotifierSecret)
	assert.NoError(t, err)
	creds := &storage.AWSSecurityHub_Credentials{}
	err = creds.Unmarshal([]byte(decrypted))
	fmt.Println(creds)
}
