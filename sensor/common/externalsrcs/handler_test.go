package externalsrcs

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestExternalSrcsHandler(t *testing.T) {
	handler := newHandler()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, handler.Start())

	vs := handler.ExternalSrcsValueStream().Iterator(true)

	assert.Nil(t, vs.Value())
	assert.Nil(t, vs.TryNext())

	// First message
	req := central.MsgToSensor_builder{
		PushNetworkEntitiesRequest: central.PushNetworkEntitiesRequest_builder{
			Entities: []*storage.NetworkEntityInfo{
				storage.NetworkEntityInfo_builder{
					Id:   "1",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.0.0.0/8"),
					}.Build(),
				}.Build(),
			},
			SeqID: 1,
		}.Build(),
	}.Build()
	require.NoError(t, handler.ProcessMessage(t.Context(), req))

	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Networks(), 5)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())

	// New message with 2 entities
	req = central.MsgToSensor_builder{
		PushNetworkEntitiesRequest: central.PushNetworkEntitiesRequest_builder{
			Entities: []*storage.NetworkEntityInfo{
				storage.NetworkEntityInfo_builder{
					Id:   "2",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.16.0.0/16"),
					}.Build(),
				}.Build(),
				storage.NetworkEntityInfo_builder{
					Id:   "1",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.0.0.0/8"),
					}.Build(),
				}.Build(),
			},
			SeqID: 2,
		}.Build(),
	}.Build()
	require.NoError(t, handler.ProcessMessage(t.Context(), req))

	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Networks(), 10)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())

	// New message with new entities whose CIDR evaluate to same as previous push; entities saved however CIDRs not pushed downstream.
	req = central.MsgToSensor_builder{
		PushNetworkEntitiesRequest: central.PushNetworkEntitiesRequest_builder{
			Entities: []*storage.NetworkEntityInfo{
				storage.NetworkEntityInfo_builder{
					Id:   "3",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.16.0.0/16"),
					}.Build(),
				}.Build(),
				storage.NetworkEntityInfo_builder{
					Id:   "4",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.0.0.0/8"),
					}.Build(),
				}.Build(),
			},
			SeqID: 3,
		}.Build(),
	}.Build()
	require.NoError(t, handler.ProcessMessage(t.Context(), req))

	require.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	require.Nil(t, vs.TryNext())

	// New request with one ipv4 and one ipv6 network.
	req = central.MsgToSensor_builder{
		PushNetworkEntitiesRequest: central.PushNetworkEntitiesRequest_builder{
			Entities: []*storage.NetworkEntityInfo{
				storage.NetworkEntityInfo_builder{
					Id:   "2",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.16.0.0/16"),
					}.Build(),
				}.Build(),
				storage.NetworkEntityInfo_builder{
					Id:   "1",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("2001:4860:4860::8888/32"),
					}.Build(),
				}.Build(),
			},
			SeqID: 4,
		}.Build(),
	}.Build()
	require.NoError(t, handler.ProcessMessage(t.Context(), req))

	assert.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Networks(), 5)
	assert.Len(t, vs.Value().GetIpv6Networks(), 17)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())

	// New request with outdated sequence ID; must be discarded.
	pner := &central.PushNetworkEntitiesRequest{}
	pner.SetSeqID(1)
	req = &central.MsgToSensor{}
	req.SetPushNetworkEntitiesRequest(proto.ValueOrDefault(pner))
	require.NoError(t, handler.ProcessMessage(t.Context(), req))

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())
}

func TestExternalSourcesLookup(t *testing.T) {
	handler := newHandler()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, handler.Start())

	vs := handler.ExternalSrcsValueStream().Iterator(true)

	assert.Nil(t, vs.Value())
	assert.Nil(t, vs.TryNext())

	// First message
	req := central.MsgToSensor_builder{
		PushNetworkEntitiesRequest: central.PushNetworkEntitiesRequest_builder{
			Entities: []*storage.NetworkEntityInfo{
				storage.NetworkEntityInfo_builder{
					Id:   "1",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.0.0.0/8"),
					}.Build(),
				}.Build(),
				storage.NetworkEntityInfo_builder{
					Id:   "2",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.10.0.0/16"),
					}.Build(),
				}.Build(),
				storage.NetworkEntityInfo_builder{
					Id:   "3",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("192.254.0.0/12"),
					}.Build(),
				}.Build(),
				storage.NetworkEntityInfo_builder{
					Id:   "4",
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
						Cidr: proto.String("10.0.0.0/0"),
					}.Build(),
				}.Build(),
			},
			SeqID: 1,
		}.Build(),
	}.Build()
	require.NoError(t, handler.ProcessMessage(t.Context(), req))
	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))

	expected := req.GetPushNetworkEntitiesRequest().GetEntities()[1]
	protoassert.Equal(t, expected, handler.LookupByNetwork(net.IPNetworkFromCIDRBytes([]byte{192, 10, 0, 0, 16})))

	expected = req.GetPushNetworkEntitiesRequest().GetEntities()[3]
	protoassert.Equal(t, expected, handler.LookupByNetwork(net.IPNetworkFromCIDRBytes([]byte{0, 0, 0, 0, 0})))
}
