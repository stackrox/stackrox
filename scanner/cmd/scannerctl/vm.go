package main

import (
	_ "embed"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

const mockDigest = "registry/repository@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"

//go:embed fixtures/indexContents.json
var contentsJson []byte

// scanCmd creates the scan command.
func scanVM(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "vm",
		Short: "Perform vulnerability scans.",
		Args:  cobra.ExactArgs(0),
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		digest, err := name.NewDigest(mockDigest)
		if err != nil {
			fmt.Printf("Failed to parse digest: %v\n", err)
		}

		var contents v4.Contents
		if err := jsonutil.JSONBytesToProto(contentsJson, &contents); err != nil {
			return fmt.Errorf("converting index contents to proto: %w", err)
		}

		vr, err := scanner.GetVulnerabilities(ctx, digest, &contents)
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		reportJSON, err := json.MarshalIndent(vr, "", "  ")
		if err != nil {
			return fmt.Errorf("decoding report: %w", err)
		}
		fmt.Println(string(reportJSON))
		return nil
	}
	return &cmd
}
