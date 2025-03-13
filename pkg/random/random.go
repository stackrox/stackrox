package random

import (
	"math/rand/v2"
)

const (
	// AlphanumericCharacters is the set of alphanumeric characters
	AlphanumericCharacters = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`

	// CaseInsensitiveAlpha is for use cases when the case is ignored
	CaseInsensitiveAlpha = `abcdefghijklmnopqrstuvwxyz`

	// HexValues is for use cases when you need a valid hex string like in image digests
	HexValues = `0123456789abcdef`
)

// GenerateString generates a random string based on the passed number of characters and the character set
func GenerateString(num int, charSet string) string {
	var str string
	for i := 0; i < num; i++ {
		randInt := rand.IntN(len(charSet))
		str += string(charSet[randInt])
	}
	return str
}
