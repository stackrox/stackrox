package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/admission-control/manager"
	"github.com/stackrox/rox/sensor/admission-control/settingswatch"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	log.Infof("StackRox Sensor Admission Control Service, version %s", version.GetMainVersion())

	utils.Must(mainCmd())
}

func mainCmd() error {
	if !features.AdmissionControlService.Enabled() {
		return errors.New("admission control service is not enabled")
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	mgr := manager.New()
	if err := mgr.Start(); err != nil {
		return errors.Wrap(err, "starting admission control manager")
	}

	if err := settingswatch.WatchK8sForSettingsUpdatesAsync(mgr.Stopped(), mgr.SettingsUpdateC()); err != nil {
		log.Errorf("Could not watch Kubernetes for settings updates: %v. Functionality might be impacted", err)
	}

	sig := <-sigC
	log.Infof("Received signal %v", sig)

	return nil
}
