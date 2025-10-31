package relay

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var log = logging.LoggerForModule()

type Relay struct {
	connectionReadTimeout time.Duration
	ctx                   context.Context
	sensorClient          sensor.VirtualMachineIndexReportServiceClient
	vsockServer           vsock.Server
	waitAfterFailedAccept time.Duration
}

func NewRelay(ctx context.Context, conn grpc.ClientConnInterface) *Relay {
	return &Relay{
		connectionReadTimeout: 10 * time.Second,
		ctx:                   ctx,
		sensorClient:          sensor.NewVirtualMachineIndexReportServiceClient(conn),
		vsockServer:           vsock.NewServer(),
		waitAfterFailedAccept: time.Second,
	}
}

func (r *Relay) Run() error {
	log.Info("Starting virtual machine relay")

	if err := r.vsockServer.Start(); err != nil {
		return errors.Wrap(err, "starting vsock server")
	}

	go func() {
		<-r.ctx.Done()
		r.vsockServer.Stop()
	}()

	for {
		// Accept() is blocking, but it will return when ctx is cancelled and the above goroutine calls r.vsockServer.Stop()
		conn, err := r.vsockServer.Accept()
		if err != nil {
			if r.ctx.Err() != nil {
				log.Info("Stopping virtual machine relay")
				return r.ctx.Err()
			}

			// We deliberately don't kill the listener on errors. The only way to stop that is to cancel the context.
			// If we had return here on fatal errors, then compliance would continue working without the relay
			// and that would make it an invisible problem to the user.
			log.Errorf("Error accepting connection: %v", err)

			time.Sleep(r.waitAfterFailedAccept) // Prevent a tight loop
			continue
		}
		metrics.VsockConnectionsAccepted.Inc()

		if err := r.vsockServer.AcquireSemaphore(r.ctx); err != nil {
			if r.ctx.Err() != nil {
				log.Info("Stopping virtual machine relay")
				return r.ctx.Err()
			}

			log.Warnf("Failed to acquire semaphore to handle connection: %v", err)

			// When the concurrency limit is reached, the semaphore cannot be acquired. We close the connection and
			// continue to listen. In this case, there is no need to add an extra wait to prevent a busy loop, because
			// we already waited semaphoreTimeout
			if err := conn.Close(); err != nil {
				log.Warnf("Failed to close connection after failing to acquire semaphore: %v", err)
			}

			continue
		}

		go r.handleConnection(conn)
	}
}

func (r *Relay) handleConnection(conn net.Conn) {
	defer r.vsockServer.ReleaseSemaphore()

	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()

	indexReport, err := vsock.HandleConnection(conn, r.connectionReadTimeout)
	if err != nil {
		log.Errorf("Error handling vsock connection from %v: %v", conn.RemoteAddr(), err)
		return
	}

	if err = sendReportToSensor(r.ctx, indexReport, r.sensorClient); err != nil {
		log.Debugf("Error sending index report to sensor (vsock CID: %s): %v", indexReport.GetVsockCid(), err)
		log.Errorf("Error sending report to sensor (vsock CID: %s): %v", indexReport.GetVsockCid(), err)
	}
}
