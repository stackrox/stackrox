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

type fakeNetworkFlowManager struct {
	stopper          concurrency.Stopper
	ctxCancel        context.CancelFunc
	receivedMessageC chan *sensor.NetworkFlowsControlMessage
	messageToSendC   chan *sensor.NetworkConnectionInfoMessage
	receivedErrC     chan error
	sendErrC         chan error
	conn             *grpc.ClientConn
}

func newFakeNetworkFlowManager(stopper concurrency.Stopper) *fakeNetworkFlowManager {
	return &fakeNetworkFlowManager{
		stopper:          stopper,
		receivedMessageC: make(chan *sensor.NetworkFlowsControlMessage),
		receivedErrC:     make(chan error),
		sendErrC:         make(chan error),
		messageToSendC:   make(chan *sensor.NetworkConnectionInfoMessage),
	}
}

func (m *fakeNetworkFlowManager) runRecv(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoClient, msgC chan<- *sensor.NetworkFlowsControlMessage, errC chan<- error) {
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

func (m *fakeNetworkFlowManager) send(msg *sensor.NetworkConnectionInfoMessage) {
	select {
	case <-m.stopper.Flow().StopRequested():
		return
	case m.messageToSendC <- msg:
	}
}

func (m *fakeNetworkFlowManager) runSend(stream sensor.NetworkConnectionInfoService_PushNetworkConnectionInfoClient, msgC <-chan *sensor.NetworkConnectionInfoMessage, errC chan<- error) {
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

func (m *fakeNetworkFlowManager) run() {
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

func (m *fakeNetworkFlowManager) start(address string) error {
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
	cli := sensor.NewNetworkConnectionInfoServiceClient(conn)
	client, err := cli.PushNetworkConnectionInfo(ctx)
	if err != nil {
		return err
	}
	go m.runRecv(client, m.receivedMessageC, m.receivedErrC)
	go m.runSend(client, m.messageToSendC, m.sendErrC)
	go m.run()
	return nil
}
