package collector

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type fakeSignalManager struct {
	stopper          concurrency.Stopper
	ctxCancel        context.CancelFunc
	receivedMessageC chan *v1.Empty
	messageToSendC   chan *sensor.SignalStreamMessage
	receivedErrC     chan error
	sendErrC         chan error
	conn             *grpc.ClientConn
}

func newSignalManager(stopper concurrency.Stopper) *fakeSignalManager {
	return &fakeSignalManager{
		stopper:          stopper,
		receivedMessageC: make(chan *v1.Empty),
		receivedErrC:     make(chan error),
		sendErrC:         make(chan error),
		messageToSendC:   make(chan *sensor.SignalStreamMessage),
	}
}

func (m *fakeSignalManager) runRecv(stream sensor.SignalService_PushSignalsClient, msgC chan<- *v1.Empty, errC chan<- error) {
	defer close(errC)
	defer close(msgC)
	for {
		msg, err := stream.Recv()
		if err != nil {
			errC <- err
			return
		}

		select {
		case <-stream.Context().Done():
			return
		case msgC <- msg:
		}
	}
}

func (m *fakeSignalManager) send(msg *sensor.SignalStreamMessage) {
	select {
	case <-m.stopper.Flow().StopRequested():
		return
	case m.messageToSendC <- msg:
	}
}

func (m *fakeSignalManager) runSend(stream sensor.SignalService_PushSignalsClient, msgC <-chan *sensor.SignalStreamMessage, errC chan<- error) {
	defer close(errC)
	for {
		select {
		case <-stream.Context().Done():
			return
		case msg, ok := <-msgC:
			if !ok {
				errC <- errors.New("channel closed")
				return
			}
			if err := stream.Send(msg); err != nil {
				errC <- err
				return
			}
		}
	}
}

func (m *fakeSignalManager) run() {
	defer func() {
		m.ctxCancel()
		m.stopper.Client().Stop()
		close(m.messageToSendC)
		if err := m.conn.Close(); err != nil {
			log.Errorf("Error closing the grpc connection %v", err)
		}
	}()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case err := <-m.sendErrC:
			log.Errorf("Error sending %v", err)
			return
		case err := <-m.receivedErrC:
			log.Errorf("Error receiving %v", err)
			return
		case msg := <-m.receivedMessageC:
			log.Infof("Received message from sensor: %v", msg)
		}
	}
}

func (m *fakeSignalManager) start(address string) error {
	clientconn.SetUserAgent("Rox Collector")
	ctx, cancel := context.WithCancel(context.Background())
	ctx = metadata.AppendToOutgoingContext(ctx, "rox-collector-hostname", "fake-collector")
	m.ctxCancel = cancel
	tlsOpts := clientconn.TLSConfigOptions{
		UseClientCert:      clientconn.MustUseClientCert,
		ServerName:         "localhost",
		CustomCertVerifier: &insecureVerifier{},
		RootCAs:            nil,
	}
	opts := clientconn.Options{
		TLS: tlsOpts,
	}
	conn, err := clientconn.GRPCConnection(ctx, mtls.SensorSubject, address, opts)
	if err != nil {
		return err
	}
	m.conn = conn
	cli := sensor.NewSignalServiceClient(conn)
	client, err := cli.PushSignals(ctx)
	if err != nil {
		return err
	}
	go m.runRecv(client, m.receivedMessageC, m.receivedErrC)
	go m.runSend(client, m.messageToSendC, m.sendErrC)
	go m.run()
	return nil
}
