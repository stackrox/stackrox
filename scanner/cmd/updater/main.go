package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/quay/zlog"

	"github.com/stackrox/rox/scanner/internal/version"
	"github.com/stackrox/rox/scanner/updater"
)

func tryExport(outputDir string) error {
	const timeout = 1 * time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := updater.Export(ctx, outputDir)
	if errors.Is(err, context.DeadlineExceeded) {
		zlog.Error(ctx).Err(err).Msg("Export timed out")
		return err
	}
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("Failed to export the vulnerabilities")
		return err
	}

	return nil
}

func main() {
	versionFlag := flag.Bool("version", false, "Print version")
	outputDir := flag.String("output-dir", "", "Output directory")
	dbConn := flag.String("db-conn", "", "Postgres connection string")
	vulnsURL := flag.String("vulns-url", "", "URL to the vulnerabilities bundle")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version.Version)
		os.Exit(0)
	}

	if *outputDir != "" {
		const maxRetries = 3

		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := tryExport(*outputDir)
			if err == nil {
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				zlog.Error(context.Background()).Err(err).Msg("Data export attempt failed; will attempt retry if within retry limits")
				continue
			}

			zlog.Error(context.Background()).Err(err).Msg("Data updater failed to update vulnerabilities")
			log.Fatal("The data export process has failed.")
		}
		log.Fatal("The data export process has failed: max retries exceeded")
		return
	}
	if *vulnsURL != "" {
		err := updater.Load(context.Background(), *dbConn, *vulnsURL)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	log.Fatal("Invalid arguments: See `updater -h`")
	return
}
