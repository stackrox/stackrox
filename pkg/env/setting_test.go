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

func TestWithStripPrefixes(t *testing.T) {
	a := assert.New(t)

	cases := map[string]struct {
		value    string
		prefixes []string
		expValue string
	}{
		"shall remove prefix if present": {
			value:    "https://example.com",
			prefixes: []string{"https://"},
			expValue: "example.com",
		},
		"shall remove one of prefixes if first of them present": {
			value:    "https://example.com",
			prefixes: []string{"https://", "http://"},
			expValue: "example.com",
		},
		"shall remove one of prefixes if second of them present": {
			value:    "http://example.com",
			prefixes: []string{"https://", "http://"},
			expValue: "example.com",
		},
		"shall not remove more than one prefix": {
			value:    "123-abc-xyz",
			prefixes: []string{"123-", "abc-", "xyz"},
			expValue: "abc-xyz",
		},
		"value should be unchanged when no prefix matches": {
			value:    "123-abc-xyz",
			prefixes: []string{"abc"},
			expValue: "123-abc-xyz",
		},
		"value should be unchanged when 0 prefixes are provided": {
			value:    "123-abc-xyz",
			prefixes: []string{},
			expValue: "123-abc-xyz",
		},
		"value should be unchanged when prefixes are provided with nil slice": {
			value:    "123-abc-xyz",
			prefixes: nil,
			expValue: "123-abc-xyz",
		},
	}

	for tname, tt := range cases {
		t.Run(tname, func(t *testing.T) {
			name := newRandomName()
			s := RegisterSetting(name, StripAnyPrefix(tt.prefixes...))
			defer unregisterSetting(name)
			a.NoError(os.Setenv(name, tt.value))

			a.Equal(tt.expValue, s.Setting())
		})
	}
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
		// This one needs to be aligned with scanner and sensor, so more common POD_NAMESPACE name was chosen.
		if k == Namespace.EnvVar() {
			continue
		}
		assert.True(t, strings.HasPrefix(k, "ROX_"), "Env var %s doesn't start with ROX_", k)
	}
}
