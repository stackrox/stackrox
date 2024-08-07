package resources

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stretchr/testify/assert"
)

var (
	testNamespace = &storage.NamespaceMetadata{
		Id:          fixtureconsts.Namespace1,
		Name:        "stackrox",
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "test cluster",
	}

	testResourceEvent = &component.ResourceEvent{
		ForwardMessages: []*central.SensorEvent{
			{
				Id:     "536e7372-4576-4011-1111-111111111111",
				Action: central.ResourceAction_SYNC_RESOURCE,
				Timing: &central.Timing{
					Dispatcher: "Namespace Dispatcher",
					Resource:   "namespace",
					Nanos:      42,
				},
				SensorHashOneof: &central.SensorEvent_SensorHash{
					SensorHash: uint64(123456789012),
				},
				Resource: &central.SensorEvent_Namespace{
					Namespace: testNamespace,
				},
			},
		},
	}

	expectedEventJSON = `{
	"id": "536e7372-4576-4011-1111-111111111111",
	"action": "SYNC_RESOURCE",
	"timing": {
		"dispatcher": "Namespace Dispatcher",
		"resource": "namespace",
		"nanos": "42"
	},
	"sensorHash": "123456789012",
	"namespace": {
		"id": "ccaaaaaa-bbbb-4011-0000-111111111111",
		"name": "stackrox",
		"clusterId": "caaaaaaa-bbbb-4011-0000-111111111111",
		"clusterName": "test cluster"
	}
}`
)

type fakeDispatcher struct{}

func (d *fakeDispatcher) ProcessEvent(_, _ interface{}, _ central.ResourceAction) *component.ResourceEvent {
	return testResourceEvent
}

func TestWrappedDispatcherProcessEventObjectEncoding(t *testing.T) {
	var buf bytes.Buffer
	testDispatcher := &dumpingDispatcher{
		writer:     &buf,
		Dispatcher: &fakeDispatcher{},
	}

	dispatchEvent := testDispatcher.ProcessEvent(nil, testNamespace, central.ResourceAction_SYNC_RESOURCE)
	protoassert.SlicesEqual(t, testResourceEvent.ForwardMessages, dispatchEvent.ForwardMessages)

	var informerMsg InformerK8sMsg
	err := json.NewDecoder(&buf).Decode(&informerMsg)
	assert.NoError(t, err)

	assert.Len(t, informerMsg.EventsOutput, 1)
	if len(informerMsg.EventsOutput) > 0 {
		assert.JSONEq(t, expectedEventJSON, informerMsg.EventsOutput[0])
	}
}
