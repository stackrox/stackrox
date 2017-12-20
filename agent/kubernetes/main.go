package main

import (
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/stack-rox/apollo/agent/kubernetes/listener"
	"bitbucket.org/stack-rox/apollo/pkg/agent"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	a := initializeAgent()

	a.Start()

	for {
		select {
		case sig := <-sigs:
			a.Logger.Infof("Caught %s signal", sig)
			a.Stop()
			a.Logger.Info("Kubernetes Agent terminated")
			return
		}
	}
}

func initializeAgent() *agent.Agent {
	a := agent.New()

	a.Listener = listener.New()

	a.Logger.Info("Kubernetes Agent Initialized")
	return a
}
