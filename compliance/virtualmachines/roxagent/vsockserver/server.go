package vsockserver

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
)

// maxConcurrentConns is the number of connections handled simultaneously.
// intentional simplification: set to 1 because the agent serves a single
// Sensor poller; raising this would require a request queue instead of
// the current reject-and-retry approach.
const maxConcurrentConns = 1

// Backoff bounds for retrying transient (non-context) Accept() errors.
const (
	minAcceptRetryDelay = 5 * time.Millisecond
	maxAcceptRetryDelay = 1 * time.Second
)

// DefaultConnDeadline bounds the entire lifetime of one accepted connection -
// TLS handshake plus request/response - so a stalled or malicious peer
// can hold at most one goroutine and one semaphore slot for this long,
// never more, and never blocks Serve's accept loop itself (see below).
// Overridable via WithConnDeadline; see that option's doc for the
// availability/DoS trade-off involved in changing it.
const DefaultConnDeadline = 30 * time.Second

// Server listens on a VSOCK port and dispatches connections to the Handler.
// tlsCfg must be non-nil in production: sensor always dials TLS, so a
// plaintext listener is unreachable. The nil path is retained only for
// testing convenience.
type Server struct {
	handler      *Handler
	tlsCfg       *tls.Config
	sem          *semaphore.Weighted // enforces at most one concurrent HandleConn
	wg           sync.WaitGroup
	connDeadline time.Duration
}

// ServerOption configures optional Server parameters.
type ServerOption func(*Server)

// WithConnDeadline overrides the per-connection deadline (TLS handshake plus
// request/response, see defaultConnDeadline). Raising it tolerates slower
// legitimate connections (e.g. under host resource contention) but also
// lets a stalled or malicious peer occupy the single in-flight-connection
// slot for longer; lowering it does the opposite. Callers should validate
// the value against sane bounds before passing it here - this type does not
// second-guess it.
func WithConnDeadline(d time.Duration) ServerOption {
	return func(s *Server) { s.connDeadline = d }
}

// NewServer creates a VSOCK server. tlsCfg should be non-nil in production.
func NewServer(handler *Handler, tlsCfg *tls.Config, opts ...ServerOption) *Server {
	s := &Server{
		handler:      handler,
		tlsCfg:       tlsCfg,
		sem:          semaphore.NewWeighted(maxConcurrentConns),
		connDeadline: DefaultConnDeadline,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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

	// acceptRetryDelay backs off on transient (non-context) Accept() errors,
	// mirroring net/http.Server, to avoid a tight busy-loop e.g. under fd exhaustion.
	var acceptRetryDelay time.Duration
	for {
		conn, err := acceptLn.Accept()
		if err != nil {
			if ctx.Err() != nil {
				s.wg.Wait()
				return
			}
			if acceptRetryDelay == 0 {
				acceptRetryDelay = minAcceptRetryDelay
			} else {
				acceptRetryDelay *= 2
			}
			if acceptRetryDelay > maxAcceptRetryDelay {
				acceptRetryDelay = maxAcceptRetryDelay
			}
			log.Errorf("Accepting connection: %v; retrying in %v", err, acceptRetryDelay)
			time.Sleep(acceptRetryDelay)
			continue
		}
		acceptRetryDelay = 0

		// Bound the whole connection - handshake included - up front, before
		// any handshake work, none of which may ever happen on this loop
		// (see serveConn/rejectConn): Accept() is the only thing this loop
		// may ever block on, or a single stalled/malicious peer could
		// prevent every subsequent connection (including the legitimate
		// one) from ever being accepted.
		_ = conn.SetDeadline(time.Now().Add(s.connDeadline))

		// TryAcquire is non-blocking, so deciding here - synchronously, in
		// accept order - keeps "first accepted wins the slot" deterministic,
		// exactly as if this whole function ran inline. Only the handshake
		// and the actual request/response, both of which can block on a
		// slow or unresponsive peer, are pushed into a goroutine.
		if s.sem.TryAcquire(1) {
			s.wg.Go(func() {
				defer s.sem.Release(1)
				s.serveConn(ctx, conn)
			})
		} else {
			s.wg.Go(func() { s.rejectConn(ctx, conn) })
		}
	}
}

// serveConn completes the TLS handshake (if any) and dispatches conn to the
// protocol handler. Called with the single in-flight-connection slot held.
func (s *Server) serveConn(ctx context.Context, conn net.Conn) {
	if !completeHandshake(ctx, conn) {
		return
	}
	s.handler.HandleConn(conn)
}

// rejectConn completes the TLS handshake (if any) and replies
// ERROR_CODE_BUSY: another connection currently holds the single
// in-flight-connection slot. Handshaking here (rather than closing the raw
// socket outright) costs a little CPU but lets Sensor tell "agent is busy,
// retry me" apart from an unexplained connection drop.
func (s *Server) rejectConn(ctx context.Context, conn net.Conn) {
	log.Warnf("Rejecting connection from %s: another request is in flight", conn.RemoteAddr())
	if !completeHandshake(ctx, conn) {
		return
	}
	defer func() { _ = conn.Close() }()
	s.handler.writeError(conn, pb.ErrorCode_ERROR_CODE_BUSY, "agent is already serving another request; retry after a backoff")
}

// completeHandshake finishes the TLS handshake on conn, if it is a TLS
// connection, logging the negotiated parameters. Returns false, having
// already closed conn, if the handshake fails; true (a no-op) for
// plaintext connections.
//
// Bounded by the deadline Serve already set on conn - not by ctx - so a
// stalled peer can never hang past connDeadline. Always called from a
// per-connection goroutine, never from Serve's accept loop: Accept() only
// wraps the socket, the handshake runs lazily on first I/O, and completing
// it eagerly (so the logged Version/CipherSuite reflect what was actually
// negotiated instead of zero values) must not block anything else.
func completeHandshake(ctx context.Context, conn net.Conn) bool {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		log.Infof("Accepted plaintext connection from %s", conn.RemoteAddr())
		return true
	}
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		defer func() { _ = conn.Close() }()
		log.Errorf("TLS handshake with %s failed: %v", conn.RemoteAddr(), err)
		return false
	}
	log.Infof("Accepted TLS connection from %s (version=0x%04x, cipher=0x%04x)",
		conn.RemoteAddr(), tlsConn.ConnectionState().Version, tlsConn.ConnectionState().CipherSuite)
	return true
}
