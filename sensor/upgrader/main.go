package main

import (
	"context"
	"flag"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/features"
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
	features.LogFeatureFlags()

	flag.Parse()

	utils.Must(mainCmd())
}

func mainCmd() error {
	upgraderCfg, err := config.Create()
	if err != nil {
		return errors.Wrap(err, "creating upgrader config")
	}

	clientconn.SetUserAgent(clientconn.Upgrader)

	upgradeCtx, err := upgradectx.Create(context.Background(), upgraderCfg)
	if err != nil {
		return errors.Wrap(err, "creating upgrade context")
	}

	// If a workflow is explicitly specified, run that end-to-end.
	if *workflow != "" {
		if err := runner.Run(upgradeCtx, *workflow); err != nil {
			return errors.Wrapf(err, "running workflow %s", *workflow)
		}
		return nil
	}

	// Else, run the metarunner.
	if err := metarunner.Run(upgradeCtx); err != nil {
		return errors.Wrap(err, "running metarunner")
	}
	return nil
}
