package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensor/common"
	"github.com/stackrox/rox/sensor/swarm/enforcer"
	"github.com/stackrox/rox/sensor/swarm/listener"
	"github.com/stackrox/rox/sensor/swarm/orchestrator"
)

func main() {
	logger := logging.LoggerForModule()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	s := common.NewSensor(
		logger,
		listener.MustCreate(),
		enforcer.MustCreate(),
		orchestrator.MustCreate(),
	)
	s.Start()

	for {
		select {
		case sig := <-sigs:
			logger.Infof("Caught %s signal", sig)
			s.Stop()
			return
		}
	}
}
