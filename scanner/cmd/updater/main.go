package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/internal/logging"
	"github.com/stackrox/rox/scanner/internal/version"
	"github.com/stackrox/rox/scanner/updater"
)

const (
	defaultManualURL = "https://raw.githubusercontent.com/stackrox/stackrox/master/scanner/updater/manual/vulns.yaml"
	logLevelEnvVar   = "STACKROX_SCANNER_V4_UPDATER_LOG_LEVEL"
	sourcesEnvVar    = "STACKROX_SCANNER_V4_UPDATER_SOURCES"
)

var (
	logLevelRaw = os.Getenv(logLevelEnvVar)
	sourcesRaw  = os.Getenv(sourcesEnvVar)
)

func initializeLogging() error {
	level := slog.LevelInfo
	var levelErr error
	if logLevelRaw != "" {
		if err := level.UnmarshalText([]byte(logLevelRaw)); err != nil {
			level = slog.LevelInfo
			levelErr = err
		}
	}
	if err := logging.Initialize(level); err != nil {
		return err
	}
	if levelErr != nil {
		slog.Warn("invalid log level, using info", "var", logLevelEnvVar, "reason", levelErr)
	}
	return nil
}

func tryExport(ctx context.Context, outputDir string, opts *updater.ExportOptions) error {
	const timeout = 3 * time.Hour
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return updater.Export(ctx, outputDir, opts)
}

func main() {
	if err := initializeLogging(); err != nil {
		slog.Error("failed to initialize logging", "reason", err)
		os.Exit(1)
	}

	sources := config.NormalizeStringList(strings.Split(sourcesRaw, ","))
	if len(sourcesRaw) > 0 && len(sources) == 0 {
		slog.Error("unable to parse sources", "raw_sources", sourcesRaw)
		os.Exit(1)
	}

	var ctx = context.Background()

	var rootCmd = &cobra.Command{
		Use:           "updater",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "StackRox Scanner vulnerability updater",
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
			for attempt := 1; attempt <= retries; attempt++ {
				slog.InfoContext(ctx, "exporting vulnerabilities",
					"attempt", attempt,
					"manual_vulns_url", manualURL,
					"output_directory", outputDir)
				err := tryExport(ctx, outputDir, &updater.ExportOptions{
					ManualVulnURL: manualURL,
					Sources:       sources,
				})
				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) {
						slog.WarnContext(ctx, "export failed; will retry if within retry limits",
							"reason", err,
							"attempt", attempt,
							"retries", retries)
						continue
					}
					return fmt.Errorf("data export failed: %w", err)
				}
				return nil
			}
			return errors.New("data export failed: max retries exceeded")
		},
	}
	exportCmd.Flags().String("manual-url", defaultManualURL, "URL to the manual vulnerability data.")

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
		slog.Error("updater failed", "reason", err)
		os.Exit(1)
	}
}
