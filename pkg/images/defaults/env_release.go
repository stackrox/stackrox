//go:build release
// +build release

package defaults

import (
	"github.com/stackrox/rox/pkg/env"
)

var (
	imageFlavorSetting = env.RegisterSetting(imageFlavorEnvName)
)
