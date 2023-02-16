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
