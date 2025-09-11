package relay

import (
	"context"
	"errors"
	"net"
	"strconv"
	"syscall"

	"github.com/stackrox/rox/compliance/virtualmachine/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// VsockRelay provides functionality to relay virtual machine index reports from vsock to sensor
type VsockRelay interface {
	// Run starts the vsock relay loop, which waits for data and immediately sends it
	Run(ctx context.Context)
}

// vsockRelayImpl implements VsockRelay
type vsockRelayImpl struct {
	service *vsock.Service
	handler Handler
	sender  ReportSender
}

// NewVsockRelay creates a new vsock relay
func NewVsockRelay(client sensor.VirtualMachineIndexReportServiceClient) VsockRelay {
	port := DefaultVsockPort
	if s := vsock.EnvVsockPort.Setting(); s != "" {
		if p, err := strconv.ParseUint(s, 10, 32); err == nil {
			port = uint32(p)
		}
	}
	return &vsockRelayImpl{
		service: vsock.NewService(port),
		handler: NewIndexReportHandler(),
		sender:  NewReportSender(client),
	}
}

// Run starts the vsock relay loop
// This waits for index reports from the server and immediately relays them to the sensor
func (v *vsockRelayImpl) Run(ctx context.Context) {
	log.Infof("Starting vsock relay")

	adapter := vsock.ConnectionHandlerFunc(func(conn net.Conn) error {
		res, err := v.handler.HandleConnection(ctx, conn)
		if err != nil {
			return err
		}
		report, ok := res.(*v1.IndexReport)
		if !ok {
			return nil
		}
		_ = v.sender.SendIndexReport(ctx, report)
		return nil
	})

	// Start service explicitly to handle unsupported environments gracefully
	runner, err := v.service.Start()
	if err != nil {
		// If vsock is not available on this host, disable relay without error spam
		if errors.Is(err, syscall.EAFNOSUPPORT) || errors.Is(err, syscall.EADDRNOTAVAIL) || errors.Is(err, syscall.ENODEV) {
			log.Infof("Vsock not available; disabling relay: %v", err)
			return
		}
		log.Errorf("Vsock service failed to start: %v", err)
		return
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(adapter)
	}()

	select {
	case <-ctx.Done():
		_ = v.service.Stop()
		return
	case err := <-errCh:
		if err != nil {
			log.Errorf("Vsock service exited: %v", err)
		}
		_ = v.service.Stop()
		return
	}
}
