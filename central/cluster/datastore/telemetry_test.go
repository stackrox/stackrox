package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter/mocks"
	"go.uber.org/mock/gomock"
)

func Test_sendProps(t *testing.T) {
	teleMock := mocks.NewMockTelemeter(gomock.NewController(t))

	// Custom gomock.Matcher for telemeter.CallOptions:
	matchOptions := func(fn func(*telemeter.CallOptions) bool) gomock.Matcher {
		return gomock.Cond(func(o any) bool {
			opts := &telemeter.CallOptions{}
			apply := (o).(telemeter.Option)
			apply(opts)
			return fn(opts)
		})
	}

	t.Run("Test custom matcher", func(t *testing.T) {
		matchUserID := func(co *telemeter.CallOptions) bool {
			return co.UserID == "testID"
		}
		if !matchOptions(matchUserID).
			Matches(telemeter.WithUserID("testID")) {
			t.Error()
		}
	})

	// Let's construct an expectation for a telemeter.Track calls for a given
	// secured cluster ID and a set of secured cluster properties.
	captureTrack := func(id string, value string) *gomock.Call {
		return teleMock.EXPECT().Track(
			// Event name:
			"Updated Secured Cluster Identity",
			// Event properties:
			nil,
			//
			// Now go the variadic ...telemeter.Option:
			//
			// Match the cluster ID:
			matchOptions(func(opts *telemeter.CallOptions) bool {
				return opts.ClientID == id
			}),
			// Whatever group:
			gomock.Any(),
			// Match the properties map:
			matchOptions(func(opts *telemeter.CallOptions) bool {
				return opts.Traits["property"] == value
			}))
	}

	cfg := &phonehome.Config{}
	cfg.SetTelemeter(teleMock, t)

	cluster1 := &storage.Cluster{Id: "id 1"}
	cluster2 := &storage.Cluster{Id: "id 2"}

	t.Run("Send properties once per cluster", func(t *testing.T) {
		first := captureTrack(cluster1.Id, "old").Times(1)
		captureTrack(cluster2.Id, "old").Times(1).After(first)

		sendProps(cfg, cluster1, map[string]any{"property": "old"})
		sendProps(cfg, cluster2, map[string]any{"property": "old"})
		sendProps(cfg, cluster1, map[string]any{"property": "old"})
		sendProps(cfg, cluster2, map[string]any{"property": "old"})
	})

	t.Run("Send updated properties once per cluster", func(t *testing.T) {
		first := captureTrack(cluster1.Id, "new").Times(1)
		captureTrack(cluster2.Id, "new").Times(1).After(first)

		sendProps(cfg, cluster1, map[string]any{"property": "new"})
		sendProps(cfg, cluster2, map[string]any{"property": "new"})
		sendProps(cfg, cluster1, map[string]any{"property": "new"})
		sendProps(cfg, cluster2, map[string]any{"property": "new"})
	})
}
