package flags

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/env"
)

// SettingVarOpts specifies options for a settings flag variable.
type SettingVarOpts struct {
	Validator func(string) error
	Type      string
}

type settingVar struct {
	setting env.Setting
	opts    SettingVarOpts
}

func (v settingVar) Type() string {
	if v.opts.Type != "" {
		return v.opts.Type
	}
	return "string"
}

func (v settingVar) String() string {
	return v.setting.Setting()
}

func (v settingVar) Set(value string) error {
	if v.opts.Validator != nil {
		if err := v.opts.Validator(value); err != nil {
			return err
		}
	} else {
		fmt.Println("no validator")
	}
	return os.Setenv(v.setting.EnvVar(), value)
}

// ForSetting returns a pflag.Value that acts on the given setting, using default options.
func ForSetting(s env.Setting) pflag.Value {
	return ForSettingWithOptions(s, SettingVarOpts{})
}

// ForSettingWithOptions returns a pflag.Value that acts on the given setting with the specified options.
func ForSettingWithOptions(s env.Setting, opts SettingVarOpts) pflag.Value {
	return settingVar{
		setting: s,
		opts:    opts,
	}
}
