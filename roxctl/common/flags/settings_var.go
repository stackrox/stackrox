package flags

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/roxctl/common/logger"
)

// SettingVarOpts specifies options for a settings flag variable.
type SettingVarOpts struct {
	Validator func(string) error
	Type      string
}

type settingVar struct {
	setting env.Setting
	log     logger.Logger
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
		v.log.PrintfLn("no validator")
	}
	return errors.Wrap(os.Setenv(v.setting.EnvVar(), value), "could not set env")
}

// ForSetting returns a pflag.Value that acts on the given setting, using default options.
func ForSetting(s env.Setting, log logger.Logger) pflag.Value {
	return ForSettingWithOptions(s, SettingVarOpts{}, log)
}

// ForSettingWithOptions returns a pflag.Value that acts on the given setting with the specified options.
func ForSettingWithOptions(s env.Setting, opts SettingVarOpts, log logger.Logger) pflag.Value {
	return settingVar{
		setting: s,
		opts:    opts,
		log:     log,
	}
}
