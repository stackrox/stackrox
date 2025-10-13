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

	t.Setenv("CI", "true")
	SetUserAgent("abc")
	ua = GetUserAgent()
	assert.True(t, strings.Contains(ua, " CI"))
}
