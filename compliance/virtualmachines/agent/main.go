package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/compliance/virtualmachines/agent/cmd"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

func main() {
	// Create a context that is cancellable on the usual command line signals. Double
	// signal forcefully exits.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigC
		log.Errorf("%s caught, shutting down...", sig)
		// Cancel the main context.
		cancel()
		go func() {
			// A second signal will forcefully quit.
			<-sigC
			os.Exit(1)
		}()
	}()
	if err := cmd.RootCmd(ctx).Execute(); err != nil {
		log.Fatal(err)
	}
}
