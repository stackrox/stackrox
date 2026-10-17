package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

//nolint:forbidigo // Commands accept injectable streams; the process entry point supplies standard I/O.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	code, err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ci-select:", err)
	}
	os.Exit(code)
}
