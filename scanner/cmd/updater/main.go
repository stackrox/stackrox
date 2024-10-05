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

func tryExport(ctx context.Context, outputDir string, opts *updater.ExportOptions) error {
	const timeout = 2 * time.Hour
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := updater.Export(ctx, outputDir, opts)
	if err != nil {
		return err
	}
	return nil
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
		Use:   "export [--split] [--manual-url <url>] <output-dir>",
		Short: "Export vulnerabilities and write bundle(s) to <output-dir>.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputDir := args[0]
			split, err := cmd.Flags().GetBool("split")
			if err != nil {
				return err
			}
			manualURL, err := cmd.Flags().GetString("manual-url")
			if err != nil {
				return err
			}
			const retries = 3
			for attempt := 1; attempt <= retries; attempt++ {
				zlog.Info(ctx).
					Int("attempt", attempt).
					Str("manual vulns URL", manualURL).
					Str("output directory", outputDir).
					Msg("exporting vulnerabilities")
				err := tryExport(ctx, outputDir, &updater.ExportOptions{SplitBundles: split, ManualVulnURL: manualURL})
				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) {
						zlog.Warn(ctx).
							Err(err).
							Int("attempt", attempt).
							Int("retries", retries).
							Msg("export failed; will retry if within retry limits")
						continue
					}
					return fmt.Errorf("data export failed: %w", err)
				}
				return nil
			}
			return errors.New("data export failed: max retries exceeded")
		},
	}
	exportCmd.Flags().Bool("split", false,
		"If true create multiple bundles per updater, rather than a single bundle.")
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
