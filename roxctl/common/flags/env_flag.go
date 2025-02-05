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

	return flagValue
}

// flagOrConfigurationValueWithFilepathOption will either return the following:
//   - the flag value, if the flag value is not the default value (i.e. flagChanged != false).
//   - the setting's value, if the flag value is the default value (i.e. flag changed == false) _and_ the setting's value is non-empty.
//   - the configuration value (where inline values take precedence), if the flag value is the default value and the setting's value is empty (i.e. flagChanged != false).
//   - the default value, if the flag value is the default value (i.e. flag changed != false), the setting's value
//     is empty, and there is no configuration value.
func flagOrConfigurationValueWithFilepathOption(flagValue string, flagChanged bool, configInlineValue string, configInlineChanged bool, configFilePathValue string, configFilePathChanged bool, setting env.Setting) string {
	if !flagChanged {
		if setting.Setting() != "" {
			return setting.Setting()
		}

		if configFilePathChanged {
			return inlineOrFilePathValue(configInlineValue, configInlineChanged, configFilePathValue, configFilePathChanged)
		}

		if configInlineChanged {
			return configInlineValue
		}
	}

	return flagValue
}

// flagOrInlineOrFile will either return the following:
// - the inline configuration value, if an inline is provided at all, regardless if a pointer to a file is provided.
// - the pointer to the filepath, if a filepath has been provided and no inline value has been provided.
// - an empty string, if nothing is provided
// TODO: Check for edge cases
func inlineOrFilePathValue(configInlineValue string, configInlineChanged bool, configFilePathValue string, configFilePathChanged bool) string {

	if configInlineChanged {
		return configInlineValue
	}

	if configFilePathChanged && !configInlineChanged {
		return configFilePathValue
	}

	return ""
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

// booleanFlagOrConfigurationValue will return either the following:
// - the flag value, if the flag value is not the default value
func booleanFlagOrConfigurationValue(flagValue bool, flagChanged bool, configValue bool, configValueChanged bool, setting *env.BooleanSetting) bool {

	if !flagChanged {
		if setting.BooleanSetting() != setting.DefaultBooleanSetting() {
			return setting.BooleanSetting()
		}

		if configValueChanged {
			return configValue
		}
	}
	return flagValue
}
