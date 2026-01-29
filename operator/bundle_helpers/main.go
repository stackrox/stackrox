package main

import (
	"fmt"
	"os"

	"github.com/stackrox/rox/operator/bundle_helpers/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Available commands:\n")
		fmt.Fprintf(os.Stderr, "  fix-spec-descriptor-order  Fix specDescriptor ordering\n")
		fmt.Fprintf(os.Stderr, "  patch-csv                  Patch ClusterServiceVersion file (not yet implemented)\n")
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "fix-spec-descriptor-order":
		if err := cmd.FixSpecDescriptorOrder(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "patch-csv":
		fmt.Fprintf(os.Stderr, "patch-csv command not yet implemented\n")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}
