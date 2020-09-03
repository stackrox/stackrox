package stringutils

import "unicode/utf8"

// LongestCommonPrefix returns the longest common prefix of a and b, performing a byte-by-byte
// comparison. Note that this might yield invalid UTF-8 strings when applied to two valid UTF-8
// strings, as certain pairs of distinct Unicode characters share a common byte prefix in their
// UTF-8 encoding.
func LongestCommonPrefix(a, b string) string {
	prefixLen := 0
	lenA, lenB := len(a), len(b)
	for prefixLen < lenA && prefixLen < lenB {
		if a[prefixLen] != b[prefixLen] {
			break
		}
		prefixLen++
	}
	return a[:prefixLen]
}

// LongestCommonPrefixUTF8 returns the longest common prefix of a and b, correctly handling
// UTF-8 encoded Unicode characters.
func LongestCommonPrefixUTF8(a, b string) string {
	prefixLen := 0 // length of prefix in _bytes_
	lenA, lenB := len(a), len(b)
	for prefixLen < lenA && prefixLen < lenB {
		runeA, runeABytes := utf8.DecodeRuneInString(a[prefixLen:])
		runeB, _ := utf8.DecodeRuneInString(b[prefixLen:])
		if runeA != runeB {
			break
		}
		if runeA == utf8.RuneError {
			// fall back to byte comparison
			if a[prefixLen] != b[prefixLen] {
				break
			}
			prefixLen++ // advance by 1 byte
		} else {
			prefixLen += runeABytes
		}
	}
	return a[:prefixLen]
}
