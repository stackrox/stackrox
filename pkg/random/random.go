package random

import (
	"math/rand/v2"
)

// Charset is a string with available characters to use by GenerateString
type Charset string

const (
	// AlphanumericCharacters is the set of alphanumeric characters
	AlphanumericCharacters Charset = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`

	// CaseInsensitiveAlpha is for use cases when the case is ignored
	CaseInsensitiveAlpha Charset = `abcdefghijklmnopqrstuvwxyz`

	// HexValues is for use cases when you need a valid hex string like in image digests
	HexValues Charset = `0123456789abcdef`
)

// GenerateString generates a random string based on the passed number of characters and the character set
func GenerateString(num int, charSet Charset) string {
	if charSet == "" {
		return ""
	}
	var str string
	for i := 0; i < num; i++ {
		randInt := rand.IntN(len(charSet))
		str += string(charSet[randInt])
	}
	return str
}
