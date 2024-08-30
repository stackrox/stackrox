package assert

import (
	"fmt"

	"github.com/stackrox/rox/pkg/devbuild"
)

// Panicf will panic on devbuilds
func Panicf(t string, vals ...interface{}) {
	if devbuild.IsEnabled() {
		panic(fmt.Sprintf(t, vals...))
	}
}
