package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/jsonutil"
)

func metadataCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "metadata",
		Short: "Get matcher metadata.",
		Args:  cobra.ExactArgs(0),
	}
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("creating scanner client: %w", err)
		}
		metadata, err := scanner.GetMatcherMetadata(ctx)
		if err != nil {
			return fmt.Errorf("getting matcher metadata: %w", err)
		}
		json, err := jsonutil.ProtoToJSON(metadata)
		if err != nil {
			return fmt.Errorf("encoding metadata: %w", err)
		}
		fmt.Println(json)
		return nil
	}
	return &cmd
}
