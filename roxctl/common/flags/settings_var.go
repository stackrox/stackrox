package flags

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/env"
)

type settingsVar struct {
	setting env.Setting
}

func (v settingsVar) Type() string {
	return "string"
}

func (v settingsVar) String() string {
	return v.setting.Setting()
}

func (v settingsVar) Set(value string) error {
	return os.Setenv(v.setting.EnvVar(), value)
}

// ForSetting returns a pflag.Value that acts on the given setting.
func ForSetting(s env.Setting) pflag.Value {
	return settingsVar{
		setting: s,
	}
}
