package mock_test

import (
	"maps"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment/mock"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stretchr/testify/assert"
)

func TestNewMockServer(t *testing.T) {
	s, dataCh := mock.NewServer(1)
	defer s.Close()
	tm := segment.NewTelemeter("key", s.URL,
		"client", "Test", "0.0.0",
		time.Second, 1, nil)

	tm.Track("test event",
		map[string]any{
			"prop1": "value1",
		},
		telemeter.WithGroup("Group1", "gid1"),
		telemeter.WithTraits(map[string]any{
			"trait1": 42,
		}),
	)

	message := maps.Collect(mock.FilterMessageFields(
		<-dataCh,
		"type", "event", "context", "properties",
	))

	assert.Equal(t, map[string]any{
		"type":  "track",
		"event": "test event",
		"properties": map[string]any{
			"prop1": "value1",
		},
		"context": map[string]any{
			"device": map[string]any{
				"type": "Test Server",
			},
			"groups": map[string]any{
				"Group1": []any{"gid1"},
			},
			"traits": map[string]any{
				"trait1": float64(42),
			},
		},
	}, message)
}
