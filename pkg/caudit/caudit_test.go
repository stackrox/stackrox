package caudit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventsAdded(t *testing.T) {
	ctx := context.Background()
	ctx = NewContext(ctx)

	AddEvent(ctx, StatusSuccess, "Sensor event triggered image scan")
	AddEvent(ctx, StatusSuccess, "Attemping to pull metadata from 2 mirrors")
	AddEvent(ctx, StatusFailure, "Metadata pull from mirror1.local failed: fake error")
	AddEvent(ctx, StatusSuccess, "Metadata pull from mirror2.local succeeded using secret PullSec/blah")
	AddEvent(ctx, StatusSuccess, "Image indexed via 'StackRox Scanner' in cluster '123'")
	AddEvent(ctx, StatusSuccess, "Vulnerabilities matched 'StackRox Scanner'")
	AddEvent(ctx, StatusFailure, "No signatures found for image")

	go doSomething(ctx)
	go doSomethingElse(ctx)

	time.Sleep(1 * time.Second)

	events := Events(ctx)
	assert.Len(t, events, 9)

	ts := time.Now().Format(time.RFC3339)
	fmt.Printf("[%s] Scan succeeded for image `quay.io/rhacs-eng/main:1.2.3`\n", ts)
	for _, e := range events {
		fmt.Printf("   %s\n", e)
	}
}

func doSomething(ctx context.Context) {
	AddEvent(ctx, StatusSuccess, "doSomething")
}

func doSomethingElse(ctx context.Context) {
	AddEvent(ctx, StatusSuccess, "doSomethingElse")
}
