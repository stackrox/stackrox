package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/quay/zlog"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/scanner/internal/version"
	"github.com/stackrox/rox/scanner/updater"
)

const DefaultURL = "https://raw.githubusercontent.com/stackrox/stackrox/master/scanner/updater/manual/vulns.yaml"

func tryExport(ctx context.Context, outputDir string, opts *updater.ExportOptions) (*updater.ExportStatus, error) {
	const timeout = 3 * time.Hour
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return updater.Export(ctx, outputDir, opts)
}

// printStatusSummary prints a summary of the export results.
func printStatusSummary(status *updater.ExportStatus) {
	if status == nil {
		return
	}

	successCount := status.SuccessCount()
	failureCount := status.FailureCount()
	total := len(status.Updaters)

	fmt.Printf("\nExport Summary: %d/%d updaters succeeded\n", successCount, total)

	if failureCount > 0 {
		fmt.Println("\nFailed updaters:")
		for _, u := range status.Updaters {
			if u.Status == updater.StatusFailed {
				fmt.Printf("  - %s: %s\n", u.Name, u.Error)
			}
		}
	}

	if successCount > 0 && failureCount > 0 {
		fmt.Printf("\nPartial success: %d bundles written, %d failed\n", successCount, failureCount)
		fmt.Println("See status.json in output directory for full details.")
	}
}

func main() {
	var ctx = context.Background()

	var rootCmd = &cobra.Command{
		Use:          "updater",
		Version:      version.Version,
		SilenceUsage: true,
		Short:        "StackRox Scanner vulnerability updater",
	}

	var exportCmd = &cobra.Command{
		Use:   "export [--manual-url <url>] <output-dir>",
		Short: "Export vulnerabilities and write bundle(s) to <output-dir>.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputDir := args[0]
			manualURL, err := cmd.Flags().GetString("manual-url")
			if err != nil {
				return err
			}
			const retries = 3
			var lastStatus *updater.ExportStatus
			for attempt := 1; attempt <= retries; attempt++ {
				zlog.Info(ctx).
					Int("attempt", attempt).
					Str("manual vulns URL", manualURL).
					Str("output directory", outputDir).
					Msg("exporting vulnerabilities")
				status, err := tryExport(ctx, outputDir, &updater.ExportOptions{ManualVulnURL: manualURL})
				lastStatus = status
				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) {
						zlog.Warn(ctx).
							Err(err).
							Int("attempt", attempt).
							Int("retries", retries).
							Msg("export failed; will retry if within retry limits")
						continue
					}
					// Print summary before returning error (all updaters failed)
					printStatusSummary(status)
					return fmt.Errorf("data export failed: %w", err)
				}
				// Print summary on success (may include partial failures)
				printStatusSummary(status)
				if status != nil && status.HasFailures() {
					zlog.Warn(ctx).
						Int("success", status.SuccessCount()).
						Int("failed", status.FailureCount()).
						Msg("export completed with partial failures")
				}
				return nil
			}
			printStatusSummary(lastStatus)
			return errors.New("data export failed: max retries exceeded")
		},
	}
	exportCmd.Flags().String("manual-url", DefaultURL, "URL to the manual vulnerability data.")

	var importCmd = &cobra.Command{
		Use:   "import",
		Short: "Import vulnerabilities using the provided database and URL",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dbConn, _ := cmd.Flags().GetString("db-conn")
			vulnsURL, _ := cmd.Flags().GetString("vulns-url")
			if err := updater.Load(ctx, dbConn, vulnsURL); err != nil {
				return err
			}
			return nil
		},
	}
	importCmd.Flags().String("db-conn", "host=/var/run/postgresql",
		"Postgres connection string")
	importCmd.Flags().String("vulns-url",
		"https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulns.json.zst",
		"URL to the vulnerabilities bundle")

	rootCmd.AddCommand(exportCmd, importCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
