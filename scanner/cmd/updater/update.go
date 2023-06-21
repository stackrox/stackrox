package main

import (
	"context"

	"github.com/quay/zlog"
	"github.com/stackrox/stackrox/scanner/v4/updater"
)

func main() {
	ctx := context.Background()
	if err := updater.Export(ctx); err != nil {
		zlog.Error(ctx).Err(err).Send()
	}
}
