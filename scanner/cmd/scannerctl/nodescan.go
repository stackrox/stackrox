package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

func nodeScanCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "nodescan",
		Short: "Triggers a node scan for the node the target node indexer runs on",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		var ir *v4.IndexReport
		ir, err = scanner.CreateNodeIndexReport(ctx)

		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		reportJSON, err := json.MarshalIndent(ir, "", "  ")
		if err != nil {
			return fmt.Errorf("decoding report: %w", err)
		}
		fmt.Println(string(reportJSON))
		return nil
	}
	return &cmd
}
