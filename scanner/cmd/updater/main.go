package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"time"

	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/updater"
)

func tryExport(outputDir string) error {
	const timeout = 35 * time.Minute
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
	outputDir := flag.String("output-dir", "", "Output directory")
	flag.Parse()

	if *outputDir == "" {
		log.Fatal("Missing argument for the output directory.")
	}

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
}
