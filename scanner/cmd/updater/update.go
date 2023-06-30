package main

import (
	"context"
	"flag"

	"github.com/quay/zlog"
	"github.com/stackrox/stackrox/scanner/v4/updater"
)

func main() {
	if len(os.Args[1:]) == 0 {
		log.Fatal("Missing argument to the output directory.")
	}
	outputDir := os.Args[1]

	ctx := context.Background()
	if err := updater.Export(ctx, *outputDir); err != nil {
		zlog.Error(ctx).Err(err).Send()
	}
}
