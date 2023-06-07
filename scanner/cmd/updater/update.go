package main

import (
	"context"

	"github.com/quay/zlog"
	"github.com/stackrox/scanner/v4/updater"
)

func main() {
	err := updater.ExportAction()
	if err != nil {
		zlog.Error(context.Background()).Msg(err.Error())
	}
}
