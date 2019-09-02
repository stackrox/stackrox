package main

import (
	"context"
	"flag"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/upgrader/config"
	_ "github.com/stackrox/rox/sensor/upgrader/flags"
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

	upgraderCfg, err := config.Create()
	utils.Must(err)

	upgradeCtx, err := upgradectx.Create(context.Background(), upgraderCfg)
	utils.Must(err)

	utils.Must(runner.Run(upgradeCtx, *workflow))
}
