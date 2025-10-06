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

	var port = flag.Uint("port", 818, "vsock port to connect to")
	var packageCount = flag.Int("packages", 10, "number of packages to include in fake reports")
	var intervalMs = flag.Int("interval", 10000, "interval between reports in milliseconds")
	flag.Parse()

	log.Printf("Starting VM agent, connecting to vsock port %d with %d packages per report, sending every %d ms", *port, *packageCount, *intervalMs)

	// Create the fake agent
	fakeAgent := agent.NewFakeAgent(uint32(*port), *packageCount, *intervalMs)

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
