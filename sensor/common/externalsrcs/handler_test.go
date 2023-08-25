package externalsrcs

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	req := &central.MsgToSensor{
		Msg: &central.MsgToSensor_PushNetworkEntitiesRequest{
			PushNetworkEntitiesRequest: &central.PushNetworkEntitiesRequest{
				Entities: []*storage.NetworkEntityInfo{
					{
						Id:   "1",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.0.0.0/8",
								},
							},
						},
					},
				},
				SeqID: 1,
			},
		},
	}
	require.NoError(t, handler.ProcessMessage(req))

	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Networks(), 5)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())

	// New message with 2 entities
	req = &central.MsgToSensor{
		Msg: &central.MsgToSensor_PushNetworkEntitiesRequest{
			PushNetworkEntitiesRequest: &central.PushNetworkEntitiesRequest{
				Entities: []*storage.NetworkEntityInfo{
					{
						Id:   "2",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.16.0.0/16",
								},
							},
						},
					},
					{
						Id:   "1",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.0.0.0/8",
								},
							},
						},
					},
				},
				SeqID: 2,
			},
		},
	}
	require.NoError(t, handler.ProcessMessage(req))

	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Networks(), 10)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())

	// New message with new entities whose CIDR evaluate to same as previous push; entities saved however CIDRs not pushed downstream.
	req = &central.MsgToSensor{
		Msg: &central.MsgToSensor_PushNetworkEntitiesRequest{
			PushNetworkEntitiesRequest: &central.PushNetworkEntitiesRequest{
				Entities: []*storage.NetworkEntityInfo{
					{
						Id:   "3",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.16.0.0/16",
								},
							},
						},
					},
					{
						Id:   "4",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.0.0.0/8",
								},
							},
						},
					},
				},
				SeqID: 3,
			},
		},
	}
	require.NoError(t, handler.ProcessMessage(req))

	require.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	require.Nil(t, vs.TryNext())

	// New request with one ipv4 and one ipv6 network.
	req = &central.MsgToSensor{
		Msg: &central.MsgToSensor_PushNetworkEntitiesRequest{
			PushNetworkEntitiesRequest: &central.PushNetworkEntitiesRequest{
				Entities: []*storage.NetworkEntityInfo{
					{
						Id:   "2",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.16.0.0/16",
								},
							},
						},
					},
					{
						Id:   "1",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "2001:4860:4860::8888/32",
								},
							},
						},
					},
				},
				SeqID: 4,
			},
		},
	}
	require.NoError(t, handler.ProcessMessage(req))

	assert.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Networks(), 5)
	assert.Len(t, vs.Value().GetIpv6Networks(), 17)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())

	// New request with outdated sequence ID; must be discarded.
	req = &central.MsgToSensor{
		Msg: &central.MsgToSensor_PushNetworkEntitiesRequest{
			PushNetworkEntitiesRequest: &central.PushNetworkEntitiesRequest{
				SeqID: 1,
			},
		},
	}
	require.NoError(t, handler.ProcessMessage(req))

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
	req := &central.MsgToSensor{
		Msg: &central.MsgToSensor_PushNetworkEntitiesRequest{
			PushNetworkEntitiesRequest: &central.PushNetworkEntitiesRequest{
				Entities: []*storage.NetworkEntityInfo{
					{
						Id:   "1",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.0.0.0/8",
								},
							},
						},
					},
					{
						Id:   "2",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.10.0.0/16",
								},
							},
						},
					},
					{
						Id:   "3",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "192.254.0.0/12",
								},
							},
						},
					},
					{
						Id:   "4",
						Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
						Desc: &storage.NetworkEntityInfo_ExternalSource_{
							ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
								Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
									Cidr: "10.0.0.0/0",
								},
							},
						},
					},
				},
				SeqID: 1,
			},
		},
	}
	require.NoError(t, handler.ProcessMessage(req))
	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))

	expected := req.GetPushNetworkEntitiesRequest().GetEntities()[1]
	assert.Equal(t, expected, handler.LookupByNetwork(net.IPNetworkFromCIDRBytes([]byte{192, 10, 0, 0, 16})))

	expected = req.GetPushNetworkEntitiesRequest().GetEntities()[3]
	assert.Equal(t, expected, handler.LookupByNetwork(net.IPNetworkFromCIDRBytes([]byte{0, 0, 0, 0, 0})))
}
