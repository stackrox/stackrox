package pkg

import (
	"testing"
)

// TEMPORARY: Test to validate race detector works with musl-gcc builds.
// This test deliberately creates a data race to verify the race detector
// is functioning correctly in our build system.
//
// Run with: RACE=true go test -race -run TestRaceDetectorWorks -v ./pkg
//
// Expected: Test should FAIL with race detector warnings.
// To be removed after CI validation.
func TestRaceDetectorWorks(t *testing.T) {
	var counter int

	// Deliberately create a data race - one writer, one reader
	done := make(chan bool)

	go func() {
		for i := 0; i < 1000; i++ {
			counter++ // Write without synchronization
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			_ = counter // Read without synchronization
		}
		done <- true
	}()

	<-done
	<-done

	// The test logic passes, but race detector should fail it
	t.Logf("Final counter value: %d (race detector should have flagged this)", counter)
}
