package clientconn

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetUserAgent(t *testing.T) {
	ua := GetUserAgent()
	assert.True(t, strings.HasPrefix(ua, "StackRox/"))
	assert.NotContains(t, ua, "CI")

	SetUserAgent("abc")
	ua = GetUserAgent()
	assert.True(t, strings.HasPrefix(ua, "abc/"))
	assert.NotContains(t, ua, "CI")

	for ci, expected := range map[string]bool{
		"yes":   true,
		"no":    true,
		"true":  true,
		"":      true,
		"false": false,
		"False": false,
	} {
		t.Setenv("CI", ci)
		SetUserAgent("test")
		ua = GetUserAgent()
		assert.Equal(t, expected, strings.Contains(ua, " CI"))
	}
}
