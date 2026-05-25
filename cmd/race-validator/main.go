package main

import (
	"fmt"
	"time"
)

// TEMPORARY: Minimal binary to validate race detector works at runtime
// This binary deliberately creates a data race to prove that race-built
// binaries detect races when executed (not just during go test).
//
// Build with: RACE=true GOOS=linux GOARCH=amd64 scripts/go-build.sh ./cmd/race-validator
// Run: ./bin/linux_amd64/race-validator
//
// Expected: Binary should print "WARNING: DATA RACE" and exit with error
func main() {
	fmt.Println("Starting race validator...")
	fmt.Println("This binary will deliberately trigger a data race.")

	var counter int

	// Deliberately create a data race
	go func() {
		for i := 0; i < 1000; i++ {
			counter++ // Write without synchronization
		}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			_ = counter // Read without synchronization
		}
	}()

	time.Sleep(100 * time.Millisecond)

	fmt.Printf("Race validator completed. Final counter: %d\n", counter)
	fmt.Println("If race detector is working, you should see 'WARNING: DATA RACE' above")
}
