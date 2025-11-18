package relay

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/handler"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/server"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

// Handler processes connections carrying virtual machine index reports.
type Handler interface {
	Handle(ctx context.Context, conn net.Conn) error
}

// Server accepts and manages connections with concurrency control.
type Server interface {
	Run(ctx context.Context, handler server.ConnectionHandler) error
}

type Relay struct {
	handler Handler
	server  Server
}

func NewRelay(conn grpc.ClientConnInterface) (*Relay, error) {
	sensorClient := sensor.NewVirtualMachineIndexReportServiceClient(conn)

	listener, err := vsock.NewListener()
	if err != nil {
		return nil, errors.Wrap(err, "creating vsock listener")
	}

	return &Relay{
		handler: handler.New(sensorClient),
		server:  server.New(listener),
	}, nil
}

func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")
	// The server handles shutdown by closing its listener when ctx is cancelled
	return r.server.Run(ctx, r.handler.Handle)
}
