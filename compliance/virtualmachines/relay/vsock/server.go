package vsock

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/sync/semaphore"
)

var log = logging.LoggerForModule()

type Server interface {
	Accept() (net.Conn, error)
	AcquireSemaphore(ctx context.Context) error
	ReleaseSemaphore()
	Start() error
	Stop()
}

type ServerImpl struct {
	listener         *vsock.Listener
	port             uint32
	semaphore        *semaphore.Weighted
	semaphoreTimeout time.Duration
}

var _ Server = (*ServerImpl)(nil)

func NewServer() *ServerImpl {
	port := env.VirtualMachinesVsockPort.IntegerSetting()
	maxConcurrentConnections := env.VirtualMachinesMaxConcurrentVsockConnections.IntegerSetting()
	semaphoreTimeout := env.VirtualMachinesConcurrencyTimeout.DurationSetting()
	return &ServerImpl{
		port:             uint32(port),
		semaphore:        semaphore.NewWeighted(int64(maxConcurrentConnections)),
		semaphoreTimeout: semaphoreTimeout,
	}
}

func (s *ServerImpl) Accept() (net.Conn, error) {
	if s.listener == nil {
		return nil, fmt.Errorf("vsock server has not been started on port %d", s.port)
	}
	return s.listener.Accept()
}

func (s *ServerImpl) AcquireSemaphore(parentCtx context.Context) error {
	semCtx, cancel := context.WithTimeout(parentCtx, s.semaphoreTimeout)
	defer cancel()

	metrics.VsockSemaphoreQueueSize.Inc()
	defer metrics.VsockSemaphoreQueueSize.Dec()
	if err := s.semaphore.Acquire(semCtx, 1); err != nil {
		reason := "unknown"
		if errors.Is(err, context.DeadlineExceeded) {
			log.Debug("Could not acquire semaphore, too many concurrent vsock connections")
			reason = "concurrency_limit"
		} else if errors.Is(err, context.Canceled) {
			log.Debug("Could not acquire semaphore, the context was canceled")
			reason = "context_canceled"
		}
		metrics.VsockSemaphoreAcquisitionFailures.With(prometheus.Labels{"reason": reason}).Inc()
		return errors.Wrap(err, "failed to acquire semaphore")
	}
	metrics.VsockSemaphoreHoldingSize.Inc()
	return nil
}

func (s *ServerImpl) ReleaseSemaphore() {
	s.semaphore.Release(1)
	metrics.VsockSemaphoreHoldingSize.Dec()
}

func (s *ServerImpl) Start() error {
	log.Debugf("Starting vsock server on port %d", s.port)
	l, err := vsock.ListenContextID(vsock.Host, s.port, nil)
	if err != nil {
		return errors.Wrapf(err, "listening on port %d", s.port)
	}
	s.listener = l
	return nil
}

func (s *ServerImpl) Stop() {
	log.Infof("Stopping vsock server on port %d", s.port)
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Errorf("Error closing vsock listener: %v", err)
		}
	}
}
