package cryptoutils

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"

	"github.com/pkg/errors"
)

const gcmNonceSizeBytes = 12

// CryptoCodec interface allows encrypting and decrypting secrets using a key
type CryptoCodec interface {
	// Encrypt encrypts the given text and returns the encrypted
	// bytes as a base64 std encoded string. The encryption key should be a base64 std encoded string.
	Encrypt(keyString string, stringToEncrypt string) (string, error)

	// Decrypt decrypts the given base64 std encoded encrypted string
	// and returns the decrypted bytes as string. The encryption key should be a base64 std encoded string.
	Decrypt(keyString string, stringToDecrypt string) (string, error)
}

// NewGCMCryptoCodec returns new CryptoCodec that can perform GCM encryption/decryption
func NewGCMCryptoCodec() CryptoCodec {
	return &gcmCryptoCodecImpl{
		nonceGen: NewNonceGenerator(gcmNonceSizeBytes, nil),
	}
}

type gcmCryptoCodecImpl struct {
	nonceGen NonceGenerator
}

// Encrypt GCM encrypts the given text and returns the encrypted
// bytes as a base64 std encoded string. The encryption key should be a base64 std encoded string.
func (gcm *gcmCryptoCodecImpl) Encrypt(keyString string, stringToEncrypt string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return "", err
	}
	bytesToEncrypt := []byte(stringToEncrypt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSizeBytes)
	if err != nil {
		return "", err
	}

	nonce, err := gcm.nonceGen.NonceBytes()
	if err != nil {
		return "", err
	}

	cipherText := aesgcm.Seal(nil, nonce, bytesToEncrypt, nil)
	// Append nonce at the beginning of encrypted string so that it can be reused at decryption
	cipherText = append(nonce, cipherText...)
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
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	cipherText, err := base64.StdEncoding.DecodeString(stringToDecrypt)
	if err != nil {
		return "", err
	}
	if len(cipherText) < aesgcm.NonceSize() {
		return "", errors.New("Invalid encrypted string")
	}
	nonce := cipherText[:aesgcm.NonceSize()]
	decrypted, err := aesgcm.Open(nil, nonce, cipherText[aesgcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}
