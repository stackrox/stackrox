package common

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/pkg/sync"
)

const (
	hardQuitSigIntThreshold = 3
)

var (
	once   sync.Once
	intCtx context.Context
)

// Context returns a context that is suited for interactive use from the command line. In particular it's responsive
// to interrupts and could probably bundle in the default timeout behavior as well.
func Context() context.Context {
	once.Do(func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT)
		ctx, cancelFunc := context.WithCancel(context.Background())
		go func() {
			numSigInts := 0
			for range ch {
				cancelFunc() // ok to be called multiple times
				numSigInts++
				if numSigInts >= hardQuitSigIntThreshold {
					fmt.Fprintf(os.Stderr, "Received %d interrupt signals. Exiting immediately...\n", hardQuitSigIntThreshold)
					os.Exit(1)
				}
				fmt.Fprintf(os.Stderr, "Received %d interrupt signal(s)\n", numSigInts)
			}
		}()
		intCtx = ctx
	})
	return intCtx
}
