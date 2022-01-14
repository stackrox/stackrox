//go:build !release
// +build !release

package defaults

import (
	"os"
)

var (
	SetRoxImageFlavorEnv func() = func() {
		if _, found := os.LookupEnv(imageFlavorEnvName); !found {
			os.Setenv(imageFlavorEnvName, "development_build")
		}
	}
)
