package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeHeaderValue(t *testing.T) {
	const vchars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	for value, expected := range map[string]string{
		"--abc ***":                 "--abc ***",
		"abc\ndef\tghi\rjkl\x00mno": "abc def ghi jkl mno",
		"":                          "",
		vchars:                      vchars,
	} {
		assert.Equal(t, expected, sanitizeHeaderValue(value), value)
	}
}
