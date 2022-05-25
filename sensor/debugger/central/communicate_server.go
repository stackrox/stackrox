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

// FakeService represents a fake central gRPC that reads and sends messages to sensor's connected gRPC stream.
type FakeService struct {
	ConnectionStarted concurrency.Signal
	KillSwitch        concurrency.Signal

	stream central.SensorService_CommunicateServer

	// initialMessages are messages to be sent to sensor once connection is open
	initialMessages []*central.MsgToSensor

	// ordered messages received from sensor
	receivedMessages []*central.MsgFromSensor

	receivedLock sync.RWMutex

	messageCallback func(sensor *central.MsgFromSensor)

	t *testing.T
}

// GetAllMessages clones and returns all messages that were ingested by central gRPC.
func (s *FakeService) GetAllMessages() []*central.MsgFromSensor {
	var output []*central.MsgFromSensor
	s.receivedLock.RLock()
	defer s.receivedLock.RUnlock()
	for _, m := range s.receivedMessages {
		output = append(output, m.Clone())
	}
	return output
}

// ClearReceivedBuffer will wipe all messages read from sensor.
func (s *FakeService) ClearReceivedBuffer() {
	s.receivedLock.Lock()
	defer s.receivedLock.Unlock()
	s.receivedMessages = []*central.MsgFromSensor{}
}

// MakeFakeCentralWithInitialMessages creates a fake gRPC connection that sends `initialMessages` on startup.
// Once communicate is called and the gRPC stream is enabled, this instance will send all `initialMessages` in order.
func MakeFakeCentralWithInitialMessages(initialMessages ...*central.MsgToSensor) *FakeService {
	return &FakeService{
		ConnectionStarted: concurrency.NewSignal(),
		KillSwitch:        concurrency.NewSignal(),
		initialMessages:   initialMessages,
		receivedMessages:  []*central.MsgFromSensor{},
		receivedLock:      sync.RWMutex{},
		messageCallback: func(_ *central.MsgFromSensor) { /* noop */ },
	}
}

// BlockReceive will read from the connection stream if the connection was already established.
// Messages read here were sent from Sensor.
// The caller might get blocked because it waits until connection is available.
func (s *FakeService) BlockReceive() (*central.MsgFromSensor, error) {
	s.ConnectionStarted.Wait()
	return s.stream.Recv()
}

// BlockSend will send to the connection stream if the connection was already established.
// This message will be received by Sensor.
// The caller might get blocked because it waits until connection is available.
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

// Communicate fakes the central communicate gRPC service by sending a test only gRPC stream to sensor.
// This stream can be killed by calling `s.KillSwitch.Signal()`.
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

// OnMessage is a utility test-only function that allows the caller (test or local-sensor) to register a callback
// for each message received. This can be used to log in stdout the messages that sensor is sending to central.
func (s *FakeService) OnMessage(callback func(sensor *central.MsgFromSensor)) {
	s.messageCallback = callback
}
