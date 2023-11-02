package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/updater"
)

func tryExport(outputDir string) error {
	const timeout = 35 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := updater.Export(ctx, outputDir)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		zlog.Error(ctx).Err(ctx.Err()).Msg("Data export attempt failed; will attempt retry if within retry limits")
		return ctx.Err()
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
		if errors.Is(err, context.DeadlineExceeded) {
			continue
		} else if err == nil {
			break
		} else {
			log.Fatal("The data export process has failed.")
		}
	}
}
