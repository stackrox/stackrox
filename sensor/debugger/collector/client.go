package collector

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type fakeCollectorManager struct {
	stopper          concurrency.Stopper
	ctxCancel        context.CancelFunc
	receivedMessageC chan *sensor.MsgToCollector
	messageToSendC   chan *sensor.MsgFromCollector
	receivedErrC     chan error
	sendErrC         chan error
	conn             *grpc.ClientConn
}

func newCollectorManager(stopper concurrency.Stopper) *fakeCollectorManager {
	return &fakeCollectorManager{
		stopper:          stopper,
		receivedMessageC: make(chan *sensor.MsgToCollector),
		receivedErrC:     make(chan error),
		sendErrC:         make(chan error),
		messageToSendC:   make(chan *sensor.MsgFromCollector),
	}
}

func (m *fakeCollectorManager) runRecv(stream sensor.CollectorService_CommunicateClient, msgC chan<- *sensor.MsgToCollector, errC chan<- error) {
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

func (m *fakeCollectorManager) send(msg *sensor.MsgFromCollector) {
	select {
	case <-m.stopper.Flow().StopRequested():
		return
	case m.messageToSendC <- msg:
	}
}

func (m *fakeCollectorManager) runSend(stream sensor.CollectorService_CommunicateClient, msgC <-chan *sensor.MsgFromCollector, errC chan<- error) {
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

func (m *fakeCollectorManager) run() {
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

func (m *fakeCollectorManager) start(address string) error {
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
	cli := sensor.NewCollectorServiceClient(conn)
	client, err := cli.Communicate(ctx)
	if err != nil {
		return err
	}
	go m.runRecv(client, m.receivedMessageC, m.receivedErrC)
	go m.runSend(client, m.messageToSendC, m.sendErrC)
	go m.run()
	return nil
}
