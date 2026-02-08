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

type fakeFileAccessManager struct {
	stopper        concurrency.Stopper
	ctxCancel      context.CancelFunc
	messageToSendC chan *sensor.FileActivity
	sendErrC       chan error
	conn           *grpc.ClientConn
}

func newFakeFileAccessManager(stopper concurrency.Stopper) *fakeFileAccessManager {
	return &fakeFileAccessManager{
		stopper:        stopper,
		sendErrC:       make(chan error),
		messageToSendC: make(chan *sensor.FileActivity),
	}
}

func (m *fakeFileAccessManager) send(msg *sensor.FileActivity) {
	select {
	case <-m.stopper.Flow().StopRequested():
		return
	case m.messageToSendC <- msg:
	}
}

func (m *fakeFileAccessManager) runSend(stream sensor.FileActivityService_CommunicateClient, msgC <-chan *sensor.FileActivity, errC chan<- error) {
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

func (m *fakeFileAccessManager) run(stream sensor.FileActivityService_CommunicateClient) {
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
			// Close the stream and wait for response
			if _, err := stream.CloseAndRecv(); err != nil {
				log.Errorf("Error closing stream: %v", err)
			}
			return
		case err := <-m.sendErrC:
			log.Errorf("Error sending %v", err)
			return
		}
	}
}

func (m *fakeFileAccessManager) start(address string) error {
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
		return errors.Wrapf(err, "creating gRPC connection to %s", address)
	}
	m.conn = conn
	cli := sensor.NewFileActivityServiceClient(conn)
	client, err := cli.Communicate(ctx)
	if err != nil {
		return errors.Wrap(err, "opening file activity stream")
	}
	go m.runSend(client, m.messageToSendC, m.sendErrC)
	go m.run(client)
	return nil
}
