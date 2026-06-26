package vsockserver

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
)

// maxConcurrentConns is the number of connections handled simultaneously.
// intentional simplification: set to 1 because the agent serves a single
// Sensor poller; raising this would require a request queue instead of
// the current reject-and-retry approach.
const maxConcurrentConns = 1

// Server listens on a VSOCK port and dispatches connections to the Handler.
// tlsCfg must be non-nil in production: sensor always dials TLS, so a
// plaintext listener is unreachable. The nil path is retained only for
// testing convenience.
type Server struct {
	handler *Handler
	tlsCfg  *tls.Config
	sem     *semaphore.Weighted // enforces at most one concurrent HandleConn
	wg      sync.WaitGroup
}

// NewServer creates a VSOCK server. tlsCfg should be non-nil in production.
func NewServer(handler *Handler, tlsCfg *tls.Config) *Server {
	return &Server{handler: handler, tlsCfg: tlsCfg, sem: semaphore.NewWeighted(maxConcurrentConns)}
}

// Serve accepts connections on ln and handles each one.
// Blocks until ctx is cancelled and the in-flight handler drains.
func (s *Server) Serve(ctx context.Context, ln net.Listener) {
	var acceptLn net.Listener
	if s.tlsCfg != nil {
		log.Info("VSOCK server: TLS enabled, wrapping listener")
		acceptLn = tls.NewListener(ln, s.tlsCfg)
	} else {
		log.Info("VSOCK server: TLS disabled, accepting plaintext")
		acceptLn = ln
	}
	defer func() { _ = acceptLn.Close() }()

	go func() {
		<-ctx.Done()
		_ = acceptLn.Close()
	}()

	for {
		conn, err := acceptLn.Accept()
		if err != nil {
			if ctx.Err() != nil {
				s.wg.Wait()
				return
			}
			log.Errorf("Accepting connection: %v", err)
			continue
		}
		if tlsConn, ok := conn.(*tls.Conn); ok {
			log.Infof("Accepted TLS connection from %s (version=0x%04x, cipher=0x%04x)",
				conn.RemoteAddr(), tlsConn.ConnectionState().Version, tlsConn.ConnectionState().CipherSuite)
		} else {
			log.Infof("Accepted plaintext connection from %s", conn.RemoteAddr())
		}

		if !s.sem.TryAcquire(1) {
			log.Warnf("Rejecting connection from %s: another request is in flight", conn.RemoteAddr())
			_ = conn.Close()
			continue
		}

		// Prevent stuck connections from leaking goroutines indefinitely.
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
		s.wg.Go(func() {
			defer s.sem.Release(1)
			s.handler.HandleConn(conn)
		})
	}
}
