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
	// Configure log format with microsecond precision
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	var port = flag.Uint("port", 1024, "vsock port to connect to")
	var packageCount = flag.Int("packages", 10, "number of packages to include in fake reports")
	flag.Parse()

	log.Printf("Starting VM agent, connecting to vsock port %d with %d packages per report", *port, *packageCount)

	// Create the fake agent
	fakeAgent := agent.NewFakeAgent(uint32(*port), *packageCount)

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
