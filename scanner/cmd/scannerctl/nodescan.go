package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func nodeScanCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodescan",
		Short: "Produces a vulnerability report for a node defined by OS, arch, and package base (el8 vs el9)",
	}
	flags := cmd.PersistentFlags()
	os := flags.String("os", "", "Operating system")
	arch := flags.String("arch", "", "Architecture")
	pbase := flags.String("package_base", "", "Package Base, e.g. el9")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("create scanner client: %w", err)
		}

		vr, err := scanner.GetNodeVulnerabilities(ctx, *os, *arch, *pbase)
		if err != nil {
			return fmt.Errorf("creating report: %w", err)
		}
		reportJSON, err := json.MarshalIndent(vr, "", " ")
		if err != nil {
			return fmt.Errorf("marshalling report: %w", err)
		}
		fmt.Println(string(reportJSON))
		return nil
	}
	return cmd
}
