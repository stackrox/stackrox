//go:build !release
// +build !release

package defaults

import (
	"os"

	"github.com/stackrox/rox/pkg/env"
)

var (
	imageFlavorSetting = ensureRoxImageFlavorIsSet()
)

func ensureRoxImageFlavorIsSet() env.Setting {
	if _, found := os.LookupEnv(imageFlavorEnvName); !found {
		return env.RegisterSetting(imageFlavorEnvName, env.WithDefault(imageFlavorDevelopment))
	}
	return env.RegisterSetting(imageFlavorEnvName)
}
