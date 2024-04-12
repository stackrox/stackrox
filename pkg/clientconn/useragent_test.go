package clientconn

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSetUserAgent(t *testing.T) {
	ua := GetUserAgent()
	assert.True(t, strings.HasPrefix(ua, "StackRox/"))
	assert.Equal(t, testutils.IsRunningInCI(), strings.Contains(ua, "CI"))

	SetUserAgent("abc")
	ua = GetUserAgent()
	assert.True(t, strings.HasPrefix(ua, "abc/"))
	assert.Equal(t, testutils.IsRunningInCI(), strings.Contains(ua, "CI"))

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
		assert.Equal(t, expected, strings.Contains(ua, " CI"), ci)
	}
}
