package tests

import (
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc/metadata"
)

type fakeService struct {
	stream            central.SensorService_CommunicateServer
	connectionStarted concurrency.Signal
	killSwitch        concurrency.Signal

	// initialMessages are messages to be sent to sensor once connection is open
	initialMessages []*central.MsgToSensor

	// ordered messages received from sensor
	receivedMessages []*central.MsgFromSensor

	receivedLock sync.RWMutex

	t *testing.T
}

func (s *fakeService) GetAllMessages() []*central.MsgFromSensor {
	var output []*central.MsgFromSensor
	s.receivedLock.RLock()
	defer s.receivedLock.RUnlock()
	for _, m := range s.receivedMessages {
		output = append(output, m.Clone())
	}
	return output
}

func (s *fakeService) ClearReceivedBuffer() {
	s.receivedLock.Lock()
	defer s.receivedLock.Unlock()
	s.receivedMessages = []*central.MsgFromSensor{}
}

func makeFakeCentralWithInitialMessages(initialMessages ...*central.MsgToSensor) *fakeService {
	return &fakeService{
		connectionStarted: concurrency.NewSignal(),
		killSwitch:        concurrency.NewSignal(),
		initialMessages:   initialMessages,
		receivedMessages:  []*central.MsgFromSensor{},
		receivedLock:      sync.RWMutex{},
	}
}

func (s *fakeService) BlockReceive() (*central.MsgFromSensor, error) {
	s.connectionStarted.Wait()
	return s.stream.Recv()
}

func (s *fakeService) BlockSend(msg *central.MsgToSensor) error {
	s.connectionStarted.Wait()
	return s.stream.Send(msg)
}

func (s *fakeService) startInputIngestion() {
	for {
		// ignore gRPC stream errors for now
		msg, _ := s.BlockReceive()
		if s.killSwitch.IsDone() {
			return
		}
		s.receivedLock.Lock()
		s.receivedMessages = append(s.receivedMessages, msg)
		s.receivedLock.Unlock()
	}

}

func (s *fakeService) Communicate(stream central.SensorService_CommunicateServer) error {
	md := metautils.NiceMD{}
	md.Set(centralsensor.SensorHelloMetadataKey, "true")
	err := stream.SetHeader(metadata.MD(md))
	if err != nil {
		s.t.Errorf("setting sensor hello metadata key: %s", err)
		return err
	}

	s.connectionStarted.Signal()

	for _, msg := range s.initialMessages {
		if err := stream.Send(msg); err != nil {
			s.t.Fatalf("failed to send initial message on gRPC stream: %s", err)
		}
	}

	s.stream = stream
	go s.startInputIngestion()
	s.killSwitch.Wait()
	return nil
}
