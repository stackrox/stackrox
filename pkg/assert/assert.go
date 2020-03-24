package assert

import (
	"fmt"

	"github.com/stackrox/rox/pkg/devbuild"
)

// Panic will panic on devbuilds
func Panic(msg interface{}) {
	if devbuild.IsEnabled() {
		panic(msg)
	}
}

// Panicf will panic on devbuilds
func Panicf(t string, vals ...interface{}) {
	if devbuild.IsEnabled() {
		panic(fmt.Sprintf(t, vals...))
	}
}
