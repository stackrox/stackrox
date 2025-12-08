package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/jsonutil"
)

const mockDigest = "registry/repository@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"

//go:embed fixtures/indexReport.json
var indexReportFixture []byte

// scanCmd creates the scan command.
func scanVM(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "vm",
		Short: "Perform vulnerability scans. Index report files can be provided as input.",
		Args:  cobra.ExactArgs(0),
	}
	flags := cmd.PersistentFlags()
	indexReportFlag := flags.String(
		"index-report-file",
		"",
		"Path to the index report in json format. Defaults to 'fixtures/indexReport.json'.")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("creating scanner client: %w", err)
		}

		digest, err := name.NewDigest(mockDigest)
		if err != nil {
			return fmt.Errorf("parsing digest: %w", err)
		}

		indexReportJson := indexReportFixture
		if *indexReportFlag != "" {
			indexReportJson, err = os.ReadFile(*indexReportFlag)
			if err != nil {
				return fmt.Errorf("reading index report %q: %w", *indexReportFlag, err)
			}
		}
		var indexReport v4.IndexReport
		if err := jsonutil.JSONBytesToProto(indexReportJson, &indexReport); err != nil {
			return fmt.Errorf("encoding index report: %w", err)
		}

		vulnReport, err := scanner.GetVulnerabilities(ctx, digest, indexReport.GetContents())
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		vulnReportJson, err := jsonutil.ProtoToJSON(vulnReport)
		if err != nil {
			return fmt.Errorf("decoding vulnerability report: %w", err)
		}
		fmt.Println(vulnReportJson)
		return nil
	}
	return &cmd
}
