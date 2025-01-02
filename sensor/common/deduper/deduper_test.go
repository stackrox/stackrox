package deduper

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/sensor/common/messagestream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStream struct {
	orderedMessages []*central.MsgFromSensor
}

func (f *fakeStream) Send(msg *central.MsgFromSensor) error {
	f.orderedMessages = append(f.orderedMessages, msg)
	return nil
}

var (
	_ messagestream.SensorMessageStream = (*fakeStream)(nil)
)

func Test_DeduperParseKeyFromEvent(t *testing.T) {
	fake := new(fakeStream)

	k1, err := deduperkey.KeyFrom("Deployment:1234")
	require.NoError(t, err)

	deduperStream := NewDedupingMessageStream(fake, map[deduperkey.Key]uint64{
		k1: 0,
	}, true)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:       "1234",
				Action:   central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_Deployment{Deployment: nil},
			},
		},
	}

	msg2 := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:       "4321",
				Action:   central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_Deployment{Deployment: nil},
			},
		},
	}

	// Send event twice so it's hashed and added to the dedupermap
	require.NoError(t, deduperStream.Send(msg))
	require.NoError(t, deduperStream.Send(msg))

	// Message 2 shouldn't be in the map because it wasn't present in the original central deduper state
	require.NoError(t, deduperStream.Send(msg2))
	require.NoError(t, deduperStream.Send(msg2))

	// observedIDs := observationSet.Close()
	err = deduperStream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{},
				},
			},
		},
	})
	require.NoError(t, err)

	lastEventSent := fake.orderedMessages[len(fake.orderedMessages)-1]
	syncMessage := lastEventSent.GetEvent().GetSynced()
	require.NotNilf(t, syncMessage, "%+v", lastEventSent)

	assert.Len(t, syncMessage.UnchangedIds, 1)
	assert.Equal(t, syncMessage.UnchangedIds[0], "Deployment:1234")

}

func Test_DeduperShallNotDedupeSomeMessages(t *testing.T) {
	cases := map[string]struct {
		msg        *central.MsgFromSensor
		key        string
		wantDedupe bool
	}{
		"Identical IndexReports shall not be deduped": {
			msg: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id: "1",
						Resource: &central.SensorEvent_IndexReport{
							IndexReport: &v4.IndexReport{
								HashId:   "nodeID",
								State:    "7", // IndexFinished
								Success:  true,
								Err:      "",
								Contents: &v4.Contents{},
							},
						},
					},
				},
			},
			key:        "IndexReport:1",
			wantDedupe: false,
		},
		"Identical ProcessIndicators shall not be deduped": {
			msg: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id: "1",
						Resource: &central.SensorEvent_ProcessIndicator{
							ProcessIndicator: &storage.ProcessIndicator{
								Id:                 "1",
								DeploymentId:       "rrr",
								ContainerName:      "rrr",
								PodId:              "aaa",
								PodUid:             "aaa",
								Signal:             nil,
								ClusterId:          "abc",
								Namespace:          "ns",
								ContainerStartTime: nil,
								ImageId:            "bbb",
							},
						},
					},
				},
			},
			key:        "ProcessIndicator:1",
			wantDedupe: false,
		},
		"Identical Runtime AlertResults shall not be deduped": {
			msg: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id: "1",
						Resource: &central.SensorEvent_AlertResults{
							AlertResults: &central.AlertResults{
								DeploymentId: "aaa",
								Alerts: []*storage.Alert{{
									Id:                "1",
									Policy:            nil,
									LifecycleStage:    0,
									ClusterId:         "aaa",
									ClusterName:       "aaa",
									Namespace:         "ns",
									NamespaceId:       "aaa",
									Entity:            nil,
									Violations:        nil,
									ProcessViolation:  nil,
									Enforcement:       nil,
									Time:              nil,
									FirstOccurred:     nil,
									ResolvedAt:        nil,
									State:             0,
									PlatformComponent: false,
									EntityType:        0,
								}},
								Stage:  storage.LifecycleStage_RUNTIME,
								Source: 0,
							},
						},
					},
				},
			},
			key:        "AlertResults:1",
			wantDedupe: false,
		},
		"Identical Deploy AlertResults shall be deduped": {
			msg: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id: "1",
						Resource: &central.SensorEvent_AlertResults{
							AlertResults: &central.AlertResults{
								DeploymentId: "aaa",
								Alerts: []*storage.Alert{{
									Id:                "1",
									Policy:            nil,
									LifecycleStage:    0,
									ClusterId:         "aaa",
									ClusterName:       "aaa",
									Namespace:         "ns",
									NamespaceId:       "aaa",
									Entity:            nil,
									Violations:        nil,
									ProcessViolation:  nil,
									Enforcement:       nil,
									Time:              nil,
									FirstOccurred:     nil,
									ResolvedAt:        nil,
									State:             0,
									PlatformComponent: false,
									EntityType:        0,
								}},
								Stage:  storage.LifecycleStage_DEPLOY,
								Source: 0,
							},
						},
					},
				},
			},
			key:        "AlertResults:1",
			wantDedupe: true,
		},
		"Identical ServiceAccounts shall be deduped": { // as an example of something that should be deduped
			msg: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id: "1",
						Resource: &central.SensorEvent_ServiceAccount{
							ServiceAccount: &storage.ServiceAccount{
								Id:               "1",
								Name:             "abc",
								Namespace:        "ns",
								ClusterName:      "cluster1",
								ClusterId:        "0abcdef",
								Labels:           nil,
								Annotations:      nil,
								CreatedAt:        nil,
								AutomountToken:   false,
								Secrets:          nil,
								ImagePullSecrets: nil,
							},
						},
					},
				},
			},
			key:        "ServiceAccount:1",
			wantDedupe: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := new(fakeStream)
			k1, err := deduperkey.KeyFrom(tc.key)
			require.NoError(t, err)

			deduperStream := NewDedupingMessageStream(fake, map[deduperkey.Key]uint64{
				k1: 0,
			}, true)

			require.NoError(t, deduperStream.Send(tc.msg))
			require.NoError(t, deduperStream.Send(tc.msg))

			if tc.wantDedupe {
				// one message only - second one should be deduped
				assert.Len(t, fake.orderedMessages, 1)
			} else {
				// two messages should reach the stream
				assert.Len(t, fake.orderedMessages, 2)

			}
		})
	}
}
