package main

import (
	"fmt"
	"os"

	"github.com/stackrox/rox/operator/bundle_helpers/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args...]\n", os.Args[0])
		fmt.Fprint(os.Stderr, "Available commands:\n")
		fmt.Fprint(os.Stderr, "  fix-spec-descriptor-order  Fix specDescriptor ordering\n")
		fmt.Fprint(os.Stderr, "  patch-csv                  Patch ClusterServiceVersion file\n")
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
		if err := cmd.PatchCSV(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}
