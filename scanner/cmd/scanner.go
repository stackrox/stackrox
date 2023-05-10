package main

import (
	"context"

	"github.com/quay/zlog"
	"github.com/stackrox/scanner/v4/updater"
	"github.com/stackrox/scanner/v4/version"
)

func main() {
	// TODO: Log intro message with Scanner version.
	zlog.Info(context.Background()).
		Str("Version", version.Version).
		Str("Mode", "TODO").
		Msg("Running Scanner")

	// Step 1. Read configuration file. This will determine how to contact DB and which mode to run in
	// Step 2. Initialize API services and create ClairCore structs based on configuration settings

	err := updater.ExportAction()
	if err != nil {
		zlog.Error(context.Background()).Msg(err.Error())
	}
}
