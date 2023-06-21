package main

import (
	"context"

	"github.com/quay/zlog"
	"github.com/stackrox/scanner/v4/updater"
)

func main() {
	ctx := context.Background()
	err := updater.Export(ctx)
	if err != nil {
		zlog.Error(context.Background()).Msg(err.Error())
	}
}
