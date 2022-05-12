package env

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newRandomName() string {
	return fmt.Sprintf("TEST_VAR_%X", time.Now().UnixNano())
}

func TestWithoutDefault(t *testing.T) {
	a := assert.New(t)

	name := newRandomName()
	s := RegisterSetting(name)
	defer unregisterSetting(name)

	a.Equal(name, s.EnvVar())
	a.Empty(s.Setting())

	a.NoError(os.Setenv(name, "foobar"))
	a.Equal("foobar", s.Setting())
}

func TestWithDefault(t *testing.T) {
	a := assert.New(t)

	name := newRandomName()
	s := RegisterSetting(name, WithDefault("baz"))
	defer unregisterSetting(name)

	a.Equal("baz", s.Setting())

	a.NoError(os.Setenv(name, "qux"))
	a.Equal("qux", s.Setting())

	a.NoError(os.Setenv(name, ""))
	a.Equal("baz", s.Setting())
}

func TestWithDefaultAndAllowEmpty(t *testing.T) {
	a := assert.New(t)

	name := newRandomName()
	s := RegisterSetting(name, WithDefault("baz"), AllowEmpty())
	defer unregisterSetting(name)

	a.Equal("baz", s.Setting())

	a.NoError(os.Setenv(name, "qux"))
	a.Equal("qux", s.Setting())

	a.NoError(os.Setenv(name, ""))
	a.Empty(s.Setting())
}

func TestDurationSetting(t *testing.T) {
	a := assert.New(t)

	name := newRandomName()
	s := registerDurationSetting(name, time.Minute)
	defer unregisterSetting(name)

	a.Equal(time.Minute, s.DurationSetting())
	a.Equal("1m0s", s.Setting())

	a.NoError(os.Setenv(name, "1h"))
	a.Equal(time.Hour, s.DurationSetting())
	a.Equal("1h0m0s", s.Setting())
}

func TestSettingEnvVarsStartWithRox(t *testing.T) {
	for k := range Settings {
		// This one slipped by, too late to change it, so ignore in the test.
		if k == NotifyEveryRuntimeEvent.EnvVar() {
			continue
		}
		assert.True(t, strings.HasPrefix(k, "ROX_"), "Env var %s doesn't start with ROX_", k)
	}
}
