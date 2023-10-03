package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// EncryptAESGCM performs AES-GCM encryption on the given text and returns the encrypted
// bytes as a base64 standard encoded string. The encryption key should be a base64 standard encoded string.
func EncryptAESGCM(keyString string, stringToEncrypt string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return "", err
	}
	bytesToEncrypt := []byte(stringToEncrypt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := aesgcm.Seal(nil, nonce, bytesToEncrypt, nil)
	// Append nonce at the beginning of encrypted string so that it can be reused at decryption
	cipherText = append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// DecryptAESGCM performs AES-GCM decryption on the given base64 encoded encrypted string
// and returns the decrypted bytes as string. The encryption key should be a base64 standard encoded string.
func DecryptAESGCM(keyString string, stringToDecrypt string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return "", err
	}
	cipherText, err := base64.StdEncoding.DecodeString(stringToDecrypt)
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

	if len(cipherText) < aesgcm.NonceSize() {
		return "", errors.New("Invalid encrypted string")
	}
	nonce := cipherText[:aesgcm.NonceSize()]
	decrypted, err := aesgcm.Open(nil, nonce, cipherText[aesgcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", decrypted), nil
}
