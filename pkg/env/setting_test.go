package env

import (
	"fmt"
	"os"
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

	a.Equal(name, s.EnvVar())
	a.Empty(s.Setting())

	a.NoError(os.Setenv(name, "foobar"))
	a.Equal("foobar", s.Setting())
}

func TestWithDefault(t *testing.T) {
	a := assert.New(t)

	name := newRandomName()
	s := RegisterSetting(name, WithDefault("baz"))

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

	a.Equal(time.Minute, s.DurationSetting())
	a.Equal("1m0s", s.Setting())

	a.NoError(os.Setenv(name, "1h"))
	a.Equal(time.Hour, s.DurationSetting())
	a.Equal("1h0m0s", s.Setting())
}
