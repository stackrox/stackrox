package main

import (
	"context"
	"flag"
	"log"

	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/updater"
)

func main() {
	// Parse command-line flags
	outputDir := flag.String("output-dir", "", "Output directory")
	flag.Parse()

	// Check if outputDir flag is provided
	if *outputDir == "" {
		log.Fatal("Missing argument for the output directory.")
	}

	ctx := context.Background()
	if err := updater.Export(ctx, *outputDir); err != nil {
		zlog.Error(ctx).Err(err).Send()
	}
}
