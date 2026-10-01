//go:build test_e2e || test_e2e_vm || sql_integration || compliance || destructive || externalbackups || test_compatibility

package tests

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"
	"sync/atomic"

	"github.com/stackrox/rox/pkg/testutils"
)

var e2eTimingSequence atomic.Uint64

type e2eActivityMarker struct {
	Event      string            `json:"event"`
	SpanID     string            `json:"span_id"`
	Activity   string            `json:"activity"`
	Helper     string            `json:"helper"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// startE2ETestActivity writes a compact marker into go test's structured output.
// The CI adapter turns it into the shared e2e_timing JSONL schema using the
// timestamp and test identity supplied by go test -json.
func startE2ETestActivity(t testutils.T, category, helper string, details map[string]string) func() {
	if !e2eTimingEnabled() {
		return func() {}
	}

	spanID := fmt.Sprintf("test-activity:%d:%d", os.Getpid(), e2eTimingSequence.Add(1))
	attributes := make(map[string]string, len(details)+1)
	maps.Copy(attributes, details)
	if named, ok := t.(interface{ Name() string }); ok {
		attributes["test_name"] = named.Name()
	}
	write := func(event string) {
		data, err := json.Marshal(e2eActivityMarker{
			Event:      event,
			SpanID:     spanID,
			Activity:   category,
			Helper:     helper,
			Attributes: attributes,
		})
		if err == nil {
			t.Logf("e2e_timing_activity %s", data)
		}
	}
	write("start")
	return func() { write("end") }
}

func e2eTimingEnabled() bool {
	switch strings.ToLower(os.Getenv("E2E_TIMING_ENABLED")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
