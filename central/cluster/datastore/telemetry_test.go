package datastore

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	cache "github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter/mocks"
	"go.uber.org/mock/gomock"
)

type testClock struct {
	t time.Time
}

func (tc *testClock) Now() time.Time {
	return tc.t
}

func (tc *testClock) add(d time.Duration) {
	tc.t = tc.t.Add(d)
}

func Test_sendProps(t *testing.T) {
	var tc testClock = testClock{time.Now()}

	// Override the global cache with custom clock for testing purposes:
	clusterIdentityCache = cache.NewExpiringCacheWithClock(&tc, 1*time.Hour)

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

	t.Run("Send same properties in an hour and a second", func(t *testing.T) {
		tc.add(1 * time.Hour)
		tc.add(1 * time.Second)
		first := captureTrack(cluster1.Id, "new").Times(1)
		captureTrack(cluster2.Id, "new").Times(1).After(first)

		sendProps(cfg, cluster1, map[string]any{"property": "new"})
		sendProps(cfg, cluster2, map[string]any{"property": "new"})
		sendProps(cfg, cluster1, map[string]any{"property": "new"})
		sendProps(cfg, cluster2, map[string]any{"property": "new"})
	})
}
