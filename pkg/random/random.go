package random

import (
	"crypto/rand"
	"math/big"
)

const (
	// AlphanumericCharacters is the set of alphanumeric characters
	AlphanumericCharacters = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`

	// CaseInsensitiveAlpha is for use cases when the case is ignored
	CaseInsensitiveAlpha = `abcdefghijklmnopqrstuvwxyz`
)

func MustGenerateString(num int, charSet string) string {
	s, err := GenerateString(num, charSet)
	if err != nil {
		panic(err)
	}
	return s
}

// GenerateString generates a random string based on the passed number of characters and the character set
func GenerateString(num int, charSet string) (string, error) {
	var str string
	max := big.NewInt(int64(len(charSet)))
	for i := 0; i < num; i++ {
		randInt, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		str += string(charSet[randInt.Int64()])
	}
	return str, nil
}
