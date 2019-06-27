package common

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	once   sync.Once
	intCtx context.Context
)

// Context returns a context that is suited for interactive use from the command line. In particular it's responsive
// to interrupts and could probably bundle in the default timeout behavior as well.
func Context() context.Context {
	once.Do(func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT)
		ctx, cancelFunc := context.WithCancel(context.Background())
		go func() {
			<-ch
			cancelFunc()
		}()
		intCtx = ctx
	})
	return intCtx
}
