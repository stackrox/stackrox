package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/logwatcher/persistentlog"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/osutils"
)

var (
	log = logging.LoggerForModule()
)

func main() {

	startPersistentLogListener()

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)

	if sig == syscall.SIGHUP {
		log.Info("Restarting central")
		osutils.Restart()
	}
	log.Info("Central terminated")
}

func startPersistentLogListener() {
	persistentReader := persistentlog.NewReader()
	start, err := persistentReader.StartReader(context.Background())
	if err != nil {
		log.Errorf("Failed to start persistent log reader %v", err)
	} else if !start {
		log.Errorf("Persistent log reader did not start because persistent logs do not exist on this node")
	}
}
