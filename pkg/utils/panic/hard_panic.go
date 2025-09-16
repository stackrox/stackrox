package panic

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/debug"
)

const (
	hardPanicDelay = 5 * time.Second
)

// HardPanic is like panic, but on debug builds additionally ensures that the
// panic will cause a crash with a full goroutine dump, independently of any
// recovery handlers.
func HardPanic(v interface{}) {
	if !buildinfo.ReleaseBuild {
		trace := debug.GetLazyStacktrace(2)
		time.AfterFunc(hardPanicDelay, func() {
			panic(fmt.Sprintf("Re-triggering panic %v as unrecoverable. Original stacktrace:\n%s", v, trace))
		})
	}
	panic(v)
}
