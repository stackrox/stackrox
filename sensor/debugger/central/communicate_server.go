package central

import (
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc/metadata"
)

type FakeService struct {
	ConnectionStarted concurrency.Signal
	KillSwitch        concurrency.Signal

	stream            central.SensorService_CommunicateServer

	// initialMessages are messages to be sent to sensor once connection is open
	initialMessages []*central.MsgToSensor

	// ordered messages received from sensor
	receivedMessages []*central.MsgFromSensor

	receivedLock sync.RWMutex

	messageCallback func(sensor *central.MsgFromSensor)

	t *testing.T
}

func (s *FakeService) GetAllMessages() []*central.MsgFromSensor {
	var output []*central.MsgFromSensor
	s.receivedLock.RLock()
	defer s.receivedLock.RUnlock()
	for _, m := range s.receivedMessages {
		output = append(output, m.Clone())
	}
	return output
}

func (s *FakeService) ClearReceivedBuffer() {
	s.receivedLock.Lock()
	defer s.receivedLock.Unlock()
	s.receivedMessages = []*central.MsgFromSensor{}
}

func MakeFakeCentralWithInitialMessages(initialMessages ...*central.MsgToSensor) *FakeService {
	return &FakeService{
		ConnectionStarted: concurrency.NewSignal(),
		KillSwitch:        concurrency.NewSignal(),
		initialMessages:   initialMessages,
		receivedMessages:  []*central.MsgFromSensor{},
		receivedLock:      sync.RWMutex{},
	}
}

func (s *FakeService) BlockReceive() (*central.MsgFromSensor, error) {
	s.ConnectionStarted.Wait()
	return s.stream.Recv()
}

func (s *FakeService) BlockSend(msg *central.MsgToSensor) error {
	s.ConnectionStarted.Wait()
	return s.stream.Send(msg)
}

func (s *FakeService) startInputIngestion() {
	for {
		// ignore gRPC stream errors for now
		msg, _ := s.BlockReceive()
		if s.KillSwitch.IsDone() {
			return
		}
		s.receivedLock.Lock()
		s.receivedMessages = append(s.receivedMessages, msg)
		s.receivedLock.Unlock()
		s.messageCallback(msg.Clone())
	}

}

func (s *FakeService) Communicate(stream central.SensorService_CommunicateServer) error {
	md := metautils.NiceMD{}
	md.Set(centralsensor.SensorHelloMetadataKey, "true")
	err := stream.SetHeader(metadata.MD(md))
	if err != nil {
		s.t.Errorf("setting sensor hello metadata key: %s", err)
		return err
	}

	s.ConnectionStarted.Signal()

	for _, msg := range s.initialMessages {
		if err := stream.Send(msg); err != nil {
			s.t.Fatalf("failed to send initial message on gRPC stream: %s", err)
		}
	}

	s.stream = stream
	go s.startInputIngestion()
	s.KillSwitch.Wait()
	return nil
}

func (s *FakeService) OnMessage(callback func(sensor *central.MsgFromSensor)) {
	s.messageCallback = callback
}
