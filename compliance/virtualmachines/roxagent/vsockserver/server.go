package vsockserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

// Server listens on a VSOCK port and responds with a cached VMReport.
// In pull mode, Sensor connects via the KubeVirt API and the connection
// is proxied through virt-handler into this server.
type Server struct {
	report atomic.Pointer[v1.VMReport]
}

// NewServer creates a Server with no cached report.
func NewServer() *Server {
	return &Server{}
}

// SetReport atomically replaces the cached report.
func (s *Server) SetReport(r *v1.VMReport) {
	s.report.Store(r)
}

// HandleConn writes the current Report to conn and closes it.
func (s *Server) HandleConn(_ context.Context, conn net.Conn) error {
	defer func() { _ = conn.Close() }()

	report := s.report.Load()
	if report == nil {
		return errors.New("no report available")
	}

	data, err := proto.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshalling report: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	log.Infof("Served VM report (%d bytes) to pull client", len(data))
	return nil
}

// Serve accepts connections on ln and handles each one.
// Blocks until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, ln net.Listener) {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Errorf("VSOCK accept error: %v", err)
			continue
		}
		go func() {
			if err := s.HandleConn(ctx, conn); err != nil {
				log.Errorf("VSOCK connection handler error: %v", err)
			}
		}()
	}
}
