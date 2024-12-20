package flags

import (
	"github.com/stackrox/rox/pkg/env"
)

// flagOrSettingValue will either return the following:
//   - the flag value, if the flag value is not the default value (i.e. flagChanged != false).
//   - the setting's value, if the flag value is the default value (i.e. flag changed == false) _and_ the setting's value is non-empty.
//   - the default value, if the flag value is the default value (i.e. flag changed != false) and the setting's value
//     is empty.
func flagOrSettingValue(flagValue string, flagChanged bool, setting env.Setting) string {
	if !flagChanged {
		if setting.Setting() != "" {
			return setting.Setting()
		}
	}
	return flagValue
}

// flagOrConfigurationValue will either return the following:
//   - the flag value, if the flag value is not the default value (i.e. flagChanged != false).
//   - the setting's value, if the flag value is the default value (i.e. flag changed == false) _and_ the setting's value is non-empty.
//   - the configuration value, if the flag value is the default value and the setting's value is empty (i.e. flagChanged != false).
//   - the default value, if the flag value is the default value (i.e. flag changed != false), the setting's value
//     is empty, and there is no configuration value.
func flagOrConfigurationValue(flagValue string, flagChanged bool, configInlineValue string, configInlineChanged bool, setting env.Setting) string {
	if !flagChanged {
		if setting.Setting() != "" {
			return setting.Setting()
		}

		if configInlineChanged {
			return configInlineValue
		}
	}

		if setting.Setting() != "" {
			return setting.Setting()
		}

	}

	return flagValue
}

// booleanFlagOrSettingValue will either return the following:
// - the flag value, if the flag value is not the default value (i.e. flagChanged != false).
// - the setting's boolean value, if the flag value is the default value (i.e. flagChanged == false)
// _and_ the setting's value is not the environment variable default value.
// - the default value, if the flag value is the default value (i.e. flagChanged == false) and the setting's value is
// the environment variable default value.
func booleanFlagOrSettingValue(flagValue bool, flagChanged bool, setting *env.BooleanSetting) bool {
	if !flagChanged {
		if setting.BooleanSetting() != setting.DefaultBooleanSetting() {
			return setting.BooleanSetting()
		}
	}
	return flagValue
}
