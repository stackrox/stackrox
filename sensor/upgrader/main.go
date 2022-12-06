package main

import (
	"context"
	"flag"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/upgrader/config"
	_ "github.com/stackrox/rox/sensor/upgrader/flags"
	"github.com/stackrox/rox/sensor/upgrader/metarunner"
	"github.com/stackrox/rox/sensor/upgrader/runner"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

var (
	log = logging.LoggerForModule()

	workflow = flag.String("workflow", "", "workflow to run")
)

func main() {
	log.Infof("StackRox Sensor Upgrader, version %s", version.GetMainVersion())

	flag.Parse()

	utils.Must(mainCmd())
}

func mainCmd() error {
	upgraderCfg, err := config.Create()
	if err != nil {
		return err
	}

	clientconn.SetUserAgent(clientconn.Upgrader)

	upgradeCtx, err := upgradectx.Create(context.Background(), upgraderCfg)
	if err != nil {
		return err
	}

	// If a workflow is explicitly specified, run that end-to-end.
	if *workflow != "" {
		return runner.Run(upgradeCtx, *workflow)
	}

	// Else, run the metarunner.
	return metarunner.Run(upgradeCtx)
}
