package central

import (
	"io"
	"log"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// FakeService represents a fake central gRPC that reads and sends messages to sensor's connected gRPC stream.
type FakeService struct {
	central.UnimplementedSensorServiceServer

	ConnectionStarted concurrency.Signal
	KillSwitch        concurrency.Signal

	// Server pointer exposes the underlying gRPC server connection
	ServerPointer *grpc.Server

	// initialMessages are messages to be sent to sensor once connection is open
	initialMessages []*central.MsgToSensor

	// ordered messages received from sensor
	receivedMessages []*central.MsgFromSensor

	receivedLock sync.RWMutex

	messageCallbackLock sync.RWMutex
	messageCallback     func(sensor *central.MsgFromSensor)

	// channel to inject messages that are sent from FakeCentral to Sensor
	centralStubMessagesC chan *central.MsgToSensor

	onShutdown func()

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
		ConnectionStarted:    concurrency.NewSignal(),
		KillSwitch:           concurrency.NewSignal(),
		initialMessages:      initialMessages,
		receivedMessages:     []*central.MsgFromSensor{},
		receivedLock:         sync.RWMutex{},
		messageCallback:      func(_ *central.MsgFromSensor) { /* noop */ },
		messageCallbackLock:  sync.RWMutex{},
		centralStubMessagesC: make(chan *central.MsgToSensor, 1),
	}
}

func (s *FakeService) ingestMessageWithLock(msg *central.MsgFromSensor) {
	s.receivedLock.Lock()
	s.receivedMessages = append(s.receivedMessages, msg)
	s.receivedLock.Unlock()
	s.messageCallbackLock.RLock()
	s.messageCallback(msg)
	s.messageCallbackLock.RUnlock()
}

func (s *FakeService) startCentralStub(stream central.SensorService_CommunicateServer) {
	s.ConnectionStarted.Wait()
	for {
		select {
		case <-s.KillSwitch.Done():
			return
		case msg, more := <-s.centralStubMessagesC:
			if !more {
				return
			}
			err := stream.Send(msg)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("error sending message to sensor: %s", err)
			}
		}
	}
}

func (s *FakeService) startInputIngestion(stream central.SensorService_CommunicateServer) {
	s.ConnectionStarted.Wait()
	for {
		var msg central.MsgFromSensor
		err := stream.RecvMsg(&msg)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("grpc stream stopped: %s\n", err)
			break
		}
		if s.KillSwitch.IsDone() {
			return
		}
		go s.ingestMessageWithLock(msg.Clone())
	}

}

// OnShutdown registers a function to be called when fake central should stop.
func (s *FakeService) OnShutdown(shutdownFn func()) {
	s.onShutdown = shutdownFn
}

// StubMessage sends a fake message to Sensor through the Central<->Sensor gRPC stream.
func (s *FakeService) StubMessage(msg *central.MsgToSensor) {
	if !s.KillSwitch.IsDone() {
		s.centralStubMessagesC <- msg
	} else {
		s.t.Errorf("tried to send MsgToSensor after KillSwitch was called: %v", msg)
	}
}

// Communicate fakes the central communicate gRPC service by sending a test only gRPC stream to sensor.
// This stream can be killed by calling `s.KillSwitch.Signal()`.
func (s *FakeService) Communicate(stream central.SensorService_CommunicateServer) error {
	defer close(s.centralStubMessagesC)

	md := metautils.NiceMD{}
	md.Set(centralsensor.SensorHelloMetadataKey, "true")
	err := stream.SetHeader(metadata.MD(md))
	if err != nil {
		s.t.Errorf("setting sensor hello metadata key: %s", err)
		return err
	}

	for _, msg := range s.initialMessages {
		if err := stream.Send(msg); err != nil {
			s.t.Fatalf("failed to send initial message on gRPC stream: %s", err)
		}
	}

	s.ConnectionStarted.Signal()
	go s.startInputIngestion(stream)
	go s.startCentralStub(stream)
	s.KillSwitch.Wait()
	return nil
}

// OnMessage is a utility test-only function that allows the caller (test or local-sensor) to register a callback
// for each message received. This can be used to log in stdout the messages that sensor is sending to central.
func (s *FakeService) OnMessage(callback func(sensor *central.MsgFromSensor)) {
	s.messageCallbackLock.Lock()
	s.messageCallback = callback
	s.messageCallbackLock.Unlock()
}

// Stop kills fake central.
func (s *FakeService) Stop() {
	if s.onShutdown != nil {
		s.onShutdown()
	}
}
