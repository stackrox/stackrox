package deduper

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
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
	})

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
