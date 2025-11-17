// Package server provides a generic connection server with concurrency control.
// It accepts connections from a net.Listener, manages concurrent connection processing
// with semaphore-based limits, and delegates actual connection handling to injected handlers.
package server

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/sync/semaphore"
)

var log = logging.LoggerForModule()

// ConnectionHandler is a function that processes an accepted connection.
// Implementations should handle errors internally; returned errors are logged but don't stop the server.
type ConnectionHandler func(ctx context.Context, conn net.Conn) error

type Server interface {
	Run(ctx context.Context, handler ConnectionHandler) error
}

type server struct {
	listener              net.Listener
	semaphore             *semaphore.Weighted
	semaphoreTimeout      time.Duration
	waitAfterFailedAccept time.Duration
}

var _ Server = (*server)(nil)

// New creates a connection server. Concurrency limits are read from env vars
// VirtualMachinesMaxConcurrentVsockConnections and VirtualMachinesConcurrencyTimeout.
// The server takes ownership of the listener and closes it when Run() returns.
func New(listener net.Listener) Server {
	maxConcurrentConnections := env.VirtualMachinesMaxConcurrentVsockConnections.IntegerSetting()
	semaphoreTimeout := env.VirtualMachinesConcurrencyTimeout.DurationSetting()
	return &server{
		listener:              listener,
		semaphore:             semaphore.NewWeighted(int64(maxConcurrentConnections)),
		semaphoreTimeout:      semaphoreTimeout,
		waitAfterFailedAccept: time.Second,
	}
}

// Run accepts connections until ctx is canceled. Handler errors are logged but don't stop the server.
// Transient accept errors are retried after 1 second to avoid making failures invisible.
func (s *server) Run(ctx context.Context, handler ConnectionHandler) error {
	log.Info("Starting relay server")

	if s.listener == nil {
		return errors.New("listener is nil")
	}

	go func() {
		<-ctx.Done()
		s.stop()
	}()

	for {
		// Accept() is blocking, but it will return when ctx is cancelled and the above goroutine calls s.stop()
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				log.Info("Stopping relay server")
				return ctx.Err()
			}

			// We deliberately don't kill the listener on errors. The only way to stop that is to cancel the context.
			// If we had return here on fatal errors, then compliance would continue working without the relay
			// and that would make it an invisible problem to the user.
			log.Errorf("Error accepting connection: %v", err)

			time.Sleep(s.waitAfterFailedAccept) // Prevent a tight loop
			continue
		}
		metrics.ConnectionsAccepted.Inc()

		if err := s.acquireSemaphore(ctx); err != nil {
			if ctx.Err() != nil {
				log.Info("Stopping connection server")
				return ctx.Err()
			}

			// This log is rate-limited because when the concurrency limit is reached it is emitted every
			// semaphoreTimeout, which is user-configurable (min: 1 second).
			logging.GetRateLimitedLogger().WarnL(
				"relay semaphore timeout",
				"Failed to acquire semaphore to handle connection: %v",
				err,
			)

			// When the concurrency limit is reached, the semaphore cannot be acquired. We close the connection and
			// continue to listen. In this case, there is no need to add an extra wait to prevent a busy loop, because
			// we already waited semaphoreTimeout
			if err := conn.Close(); err != nil {
				log.Warnf("Failed to close connection after failing to acquire semaphore: %v", err)
			}

			continue
		}

		go func(conn net.Conn) {
			defer s.releaseSemaphore()

			defer func(conn net.Conn) {
				if err := conn.Close(); err != nil {
					log.Errorf("Failed to close connection: %v", err)
				}
			}(conn)

			if err := handler(ctx, conn); err != nil {
				log.Errorf("Error handling connection from %v: %v", conn.RemoteAddr(), err)
			}
		}(conn)
	}
}

func (s *server) stop() {
	log.Info("Stopping connection server")
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Errorf("Error closing listener: %v", err)
		}
	}
}

func (s *server) acquireSemaphore(parentCtx context.Context) error {
	semCtx, cancel := context.WithTimeout(parentCtx, s.semaphoreTimeout)
	defer cancel()

	metrics.SemaphoreQueueSize.Inc()
	defer metrics.SemaphoreQueueSize.Dec()
	if err := s.semaphore.Acquire(semCtx, 1); err != nil {
		reason := "unknown"
		if errors.Is(err, context.DeadlineExceeded) {
			log.Debug("Could not acquire semaphore, too many concurrent connections")
			reason = "concurrency_limit"
		} else if errors.Is(err, context.Canceled) {
			log.Debug("Could not acquire semaphore, the context was canceled")
			reason = "context_canceled"
		}
		metrics.SemaphoreAcquisitionFailures.With(prometheus.Labels{"reason": reason}).Inc()
		return errors.Wrap(err, "failed to acquire semaphore")
	}
	metrics.SemaphoreHoldingSize.Inc()
	return nil
}

func (s *server) releaseSemaphore() {
	s.semaphore.Release(1)
	metrics.SemaphoreHoldingSize.Dec()
}
