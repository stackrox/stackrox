package main

import (
	"context"
	"flag"
	"os"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensorupgrader"
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

const (
	upgraderOwnerEnvVar    = `ROX_UPGRADER_OWNER`
	upgradeProcessIDEnvVar = `ROX_UPGRADE_PROCESS_ID`
)

func main() {
	log.Infof("StackRox Sensor Upgrader, version %s", version.GetMainVersion())

	flag.Parse()

	utils.Must(mainCmd())
}

func mainCmd() error {
	// clusterID is optional and only required when fetching the bundle, not when used in standalone mode
	clusterID := os.Getenv(sensorupgrader.ClusterIDEnvVarName)
	centralEndpoint := os.Getenv(env.CentralEndpoint.EnvVar())

	log.Infof("Configuring upgrader with: "+
		"clusterID=%q, centralEndpoint=%q, "+
		"processID=%q, owner=%q, certsOnly=%t",
		clusterID, centralEndpoint,
		os.Getenv(upgradeProcessIDEnvVar), os.Getenv(upgraderOwnerEnvVar), env.UpgraderCertsOnly.BooleanSetting())

	upgraderCfg, err := config.Create(clusterID,
		centralEndpoint,
		os.Getenv(upgradeProcessIDEnvVar),
		os.Getenv(upgraderOwnerEnvVar),
		env.UpgraderCertsOnly.BooleanSetting())
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

	// Else, run the metarunner. It will get the instructions from Central which workflow to run
	return metarunner.Run(upgradeCtx)
}
