package tlsutils

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
)

var (
	// ErrNoTLSConn is the error raised (via the `OnHandshakeError` callback) if a connection is not a TLS connection.
	ErrNoTLSConn = errors.New("connection is not a TLS connection")
)

const (
	initTempDelay = 5 * time.Millisecond
	maxTempDelay  = 1 * time.Second
)

// ListenerControl encapsulates the functionality of controlling a listener, without the ability to accept incoming
// connections.
type ListenerControl interface {
	Close() error
	Addr() net.Addr
}

// ALPNDemuxConfig allows fine-grained control over the behavior of a ALPN-demultiplexing listener.
type ALPNDemuxConfig struct {
	// MaxCloseWait is the maximum time to wait for a Close to succeed before no longer accepting
	// connections on sub-listeners.
	MaxCloseWait time.Duration
	// OnHandshakeError is called whenever there is a TLS handshake error, unless nil.
	OnHandshakeError func(net.Conn, error)
	// TLSHandshakeTimeout is the maximum time allowed for the TLS handshake to finish.
	TLSHandshakeTimeout time.Duration
	// OnHandshakeComplete is called when TLS handshake is done
	OnHandshakeComplete func(conn net.Conn, proto string)
}

// ALPNDemux takes in a single listener, and demultiplexes it onto an arbitrary number of listeners based on the
// negotiated application-level protocol.
// listenersByProto maps protocol names to pointers to `net.Listener` variables. Once `ALPNDemux` returns, these
// variables will be set to the respective demultiplexed listeners (it is valid to have the address of a `net.Listener`
// variable occur multiple times as a value, in which case the resulting listeners will handle connections for all
// respective application-level protocols). All `*net.Listener` pointers must be non-nil and initially point to a
// variable containing a nil `net.Listener`, otherwise this function will panic.
// If the negotiated application-level protocol is unknown, or the client supplied no supported application-level
// protocols in the handshake, the listener for the protocol "" handles these connections. This function panics if the
// supplied map does not contain an entry for "".
// The given listener should be a TLS listener, but there is no way to enforce this. If this is used with a non-TLS
// listener (i.e., the returned connections are not `*tls.Conn`s), the `OnHandshakeError` callback is invoked with the
// connection and `ErrNoTLSConn`.
func ALPNDemux(tlsListener net.Listener, listenersByProto map[string]*net.Listener, config ALPNDemuxConfig) ListenerControl {
	if listenersByProto[""] == nil {
		panic(errors.New("no listener specified for the default/non-ALPN case"))
	}

	chanByKeyMap := make(map[*net.Listener]chan net.Conn, len(listenersByProto))
	chanByProtoMap := make(map[string]chan<- net.Conn, len(listenersByProto))
	for proto, key := range listenersByProto {
		if key == nil {
			panic(errors.New("nil value in listenersByProto map"))
		}
		if *key != nil {
			panic(errors.New("non-nil listener in listenersByProto map value"))
		}

		ch := chanByKeyMap[key]
		if ch == nil {
			ch = make(chan net.Conn)
			chanByKeyMap[key] = ch
		}
		chanByProtoMap[proto] = ch
	}

	l := &alpnDemuxListener{
		lis:     tlsListener,
		closed:  concurrency.NewErrorSignal(),
		chanMap: chanByProtoMap,
		cfg:     config,
	}
	go l.run()

	for key, ch := range chanByKeyMap {
		*key = &fromChanListener{
			alpnDemuxListener: l,
			connC:             ch,
		}
	}

	return l
}

type alpnDemuxListener struct {
	lis     net.Listener
	closed  concurrency.ErrorSignal
	chanMap map[string]chan<- net.Conn
	cfg     ALPNDemuxConfig
}

func (l *alpnDemuxListener) Addr() net.Addr {
	return l.lis.Addr()
}

func (l *alpnDemuxListener) Close() error {
	if l.cfg.MaxCloseWait != 0 {
		time.AfterFunc(l.cfg.MaxCloseWait, func() {
			l.closed.SignalWithError(fmt.Errorf("hard-closed listener after %v", l.cfg.MaxCloseWait))
		})
	}
	return l.lis.Close()
}

func (l *alpnDemuxListener) run() {
	tempDelay := time.Duration(0)
	for {
		conn, err := l.lis.Accept()
		if err != nil {
			if l.closed.IsDone() {
				return
			}

			// Taken from golang http2 server code
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = initTempDelay
				} else {
					tempDelay *= 2
				}
				if tempDelay > maxTempDelay {
					tempDelay = maxTempDelay
				}

				time.Sleep(tempDelay)
				continue
			}

			l.closed.SignalWithError(err) // error is permanent
			return
		}

		tempDelay = 0
		go l.dispatch(conn)
	}
}

func (l *alpnDemuxListener) dispatch(conn net.Conn) {
	if err := l.doDispatch(conn); err != nil {
		if l.cfg.OnHandshakeError != nil {
			l.cfg.OnHandshakeError(conn, err)
		}
	}
}

func (l *alpnDemuxListener) doDispatch(conn net.Conn) error {
	tlsConn, _ := conn.(*tls.Conn)
	if tlsConn == nil {
		return ErrNoTLSConn
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), l.cfg.TLSHandshakeTimeout,
		errors.New("TLS handshake timeout"))
	defer cancel()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return err
	}

	alp := tlsConn.ConnectionState().NegotiatedProtocol
	if callback := l.cfg.OnHandshakeComplete; callback != nil {
		go callback(tlsConn, alp)
	}
	ch := l.chanMap[alp]
	if ch == nil {
		ch = l.chanMap[""]
	}
	select {
	case ch <- tlsConn:
	case <-l.closed.Done():
	}
	return nil
}

type fromChanListener struct {
	*alpnDemuxListener
	connC <-chan net.Conn
}

func (l *fromChanListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connC:
		return conn, nil
	case <-l.alpnDemuxListener.closed.Done():
		return nil, l.alpnDemuxListener.closed.Err()
	}
}
