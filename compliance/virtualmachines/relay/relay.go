package relay

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/handler"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/server"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

type Relay struct {
	handler *handler.Handler
	server  server.Server
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
	return r.server.Run(ctx, r.handler.Handle)
}
