package cryptocodec

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	cc   CryptoCodec
)

// CryptoCodec interface allows encrypting and decrypting secrets using a key
type CryptoCodec interface {
	// Encrypt encrypts the given text and returns the encrypted
	// bytes as a base64 std encoded string. The encryption key should be a base64 std encoded string.
	Encrypt(keyString string, stringToEncrypt string) (string, error)

	// Decrypt decrypts the given base64 std encoded encrypted string
	// and returns the decrypted bytes as string. The encryption key should be a base64 std encoded string.
	Decrypt(keyString string, stringToDecrypt string) (string, error)
}

// Singleton returns singleton instance of the crypto codec
func Singleton() CryptoCodec {
	once.Do(func() {
		cc = NewGCMCryptoCodec()
	})
	return cc
}

// NewGCMCryptoCodec returns new CryptoCodec that can perform GCM encryption/decryption
func NewGCMCryptoCodec() CryptoCodec {
	return &gcmCryptoCodecImpl{}
}

type gcmCryptoCodecImpl struct{}

// Encrypt GCM encrypts the given text and returns the encrypted
// bytes as a base64 std encoded string. The encryption key should be a base64 std encoded string.
func (gcm *gcmCryptoCodecImpl) Encrypt(keyString string, stringToEncrypt string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// FIPS 140-only mode rejects cipher.NewGCM because it allows arbitrary IVs.
	aesgcm, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return "", err
	}

	cipherText := aesgcm.Seal(nil, nil, []byte(stringToEncrypt), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt decrypts the given base64 std encoded encrypted string
// and returns the decrypted bytes as string. The encryption key should be a base64 std encoded string.
func (gcm *gcmCryptoCodecImpl) Decrypt(keyString string, stringToDecrypt string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return "", err
	}

	cipherText, err := base64.StdEncoding.DecodeString(stringToDecrypt)
	if err != nil {
		return "", err
	}
	decrypted, err := aesgcm.Open(nil, nil, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}
