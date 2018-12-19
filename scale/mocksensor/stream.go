package main

import (
	"sync"

	"github.com/stackrox/rox/generated/internalapi/central"
)

type threadSafeStream struct {
	stream central.SensorService_CommunicateClient
	mutex  sync.Mutex
}

func (s *threadSafeStream) SendEvent(event *central.SensorEvent) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.stream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: event,
		},
	})
}

func (s *threadSafeStream) SendNetworkFlows(flows *central.NetworkFlowUpdate) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.stream.Send(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_NetworkFlowUpdate{
			NetworkFlowUpdate: flows,
		},
	})
}
