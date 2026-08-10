package main

import (
	"os"

	"github.com/stackrox/rox/stackrox-tls-diagnostics/cmd"
)

func main() {
	if err := cmd.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
