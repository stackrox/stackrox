package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/stackrox/vm-agent/agent"
)

func main() {
	var port = flag.Uint("port", 1024, "vsock port to connect to")
	flag.Parse()

	log.Printf("Starting VM agent, connecting to vsock port %d", *port)

	// Create the fake agent
	fakeAgent := agent.NewFakeAgent(uint32(*port))

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Run the agent
	if err := fakeAgent.Run(ctx); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	log.Println("VM agent stopped")
}
