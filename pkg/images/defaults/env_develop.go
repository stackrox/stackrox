//go:build !release
// +build !release

package defaults

import (
	"os"
)

var (
	setRoxImageFlavorEnv = func() {
		if _, found := os.LookupEnv(imageFlavorEnvName); !found {
			if os.Setenv(imageFlavorEnvName, "development_build") != nil {
				log.Panicf("Could not set %s", imageFlavorEnvName)
			}
		}
	}
)
