package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stretchr/testify/assert"
)

func TestCompatibilityFailureReportsAndWaits(t *testing.T) {
	var events []string
	err := fmt.Errorf("scan: %w", &migrations.CompatibilityError{Message: "Central upgrade blocked"})
	handleFailure(err, func(message string) error {
		events = append(events, "write")
		assert.Equal(t, "Central upgrade blocked", message)
		return errors.New("read-only path")
	}, func(delay time.Duration) {
		events = append(events, "wait")
		assert.Equal(t, 5*time.Minute, delay)
	})
	assert.Equal(t, []string{"write", "wait"}, events, "write failure must not remove the throttle")
}

func TestTransientFailureDoesNotWait(t *testing.T) {
	handleFailure(errors.New("connection failed"), func(string) error {
		t.Fatal("transient error must not use permanent-rejection handling")
		return nil
	}, func(time.Duration) { t.Fatal("unexpected wait") })
}
