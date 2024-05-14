package datastore

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter/mocks"
	"go.uber.org/mock/gomock"
)

func Test_sendProps(t *testing.T) {
	teleMock := mocks.NewMockTelemeter(gomock.NewController(t))
	captureTrack := func(id string, props map[string]any) *gomock.Call {
		return teleMock.EXPECT().Track("Updated Secured Cluster Identity",
			nil,
			// Check the cluster ID option:
			gomock.Cond(func(x any) bool {
				opts := &telemeter.CallOptions{}
				(x).(telemeter.Option)(opts)
				return opts.ClientID == id
			}),
			// Whatever group option:
			gomock.Any(),
			// Check the properties map option:
			gomock.Cond(func(x any) bool {
				opts := &telemeter.CallOptions{}
				(x).(telemeter.Option)(opts)
				return reflect.DeepEqual(props, opts.Traits)
			}))
	}

	cfg := &phonehome.Config{}
	cfg.SetTelemeter(teleMock, t)

	cluster1 := &storage.Cluster{Id: "id 1"}
	cluster2 := &storage.Cluster{Id: "id 2"}

	t.Run("Send properties once per cluster", func(t *testing.T) {
		first := captureTrack(cluster1.Id, map[string]any{"property": "old"}).Times(1)
		captureTrack(cluster2.Id, map[string]any{"property": "old"}).Times(1).After(first)

		sendProps(cfg, cluster1, map[string]any{"property": "old"})
		sendProps(cfg, cluster2, map[string]any{"property": "old"})
		sendProps(cfg, cluster1, map[string]any{"property": "old"})
		sendProps(cfg, cluster2, map[string]any{"property": "old"})
	})

	t.Run("Send updated properties once per cluster", func(t *testing.T) {
		first := captureTrack(cluster1.Id, map[string]any{"property": "new"}).Times(1)
		captureTrack(cluster2.Id, map[string]any{"property": "new"}).Times(1).After(first)

		sendProps(cfg, cluster1, map[string]any{"property": "new"})
		sendProps(cfg, cluster2, map[string]any{"property": "new"})
		sendProps(cfg, cluster1, map[string]any{"property": "new"})
		sendProps(cfg, cluster2, map[string]any{"property": "new"})
	})
}
