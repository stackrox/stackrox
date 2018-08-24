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
	s := NewSetting(name)

	a.Equal(name, s.EnvVar())
	a.Empty(s.Setting())

	os.Setenv(name, "foobar")
	a.Equal("foobar", s.Setting())
}

func TestWithDefault(t *testing.T) {
	a := assert.New(t)

	name := newRandomName()
	s := NewSetting(name, WithDefault("baz"))

	a.Equal("baz", s.Setting())

	os.Setenv(name, "qux")
	a.Equal("qux", s.Setting())

	os.Setenv(name, "")
	a.Equal("baz", s.Setting())
}

func TestWithDefaultAndAllowEmpty(t *testing.T) {
	a := assert.New(t)

	name := newRandomName()
	s := NewSetting(name, WithDefault("baz"), AllowEmpty())

	a.Equal("baz", s.Setting())

	os.Setenv(name, "qux")
	a.Equal("qux", s.Setting())

	os.Setenv(name, "")
	a.Empty(s.Setting())
}
