package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"google.golang.org/grpc/credentials"
)

// credsFromConn is a gRPC `credentials.TransportCredentials` implementation that obtains authentication data from a
// given, existing TLS connection (or nothing if no TLS connection is used).
// Usually, `(*grpc.Server).Serve` expects a raw TCP listener, which is then wrapped into a `tls.Listener` by means of
// the transport credentials returned by `credentials.NewTLS`. This struct makes it possible to invoke `Serve` with a
// TLS listener.
type credsFromConn struct{}

func (c credsFromConn) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "tls",
		SecurityVersion:  "1.2",
	}
}

func (c credsFromConn) ClientHandshake(_ context.Context, _ string, _ net.Conn) (net.Conn, credentials.AuthInfo, error) {
	return nil, nil, errors.New("server use only")
}

func (c credsFromConn) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	tlsConn, _ := rawConn.(interface {
		net.Conn
		Handshake() error
		ConnectionState() tls.ConnectionState
	})
	if tlsConn == nil {
		return rawConn, nil, nil
	}
	if err := tlsConn.Handshake(); err != nil {
		log.Debugf("TLS handshake error from %q: %v", rawConn.RemoteAddr(), err)
		return nil, nil, err
	}
	return tlsConn, credentials.TLSInfo{State: tlsConn.ConnectionState()}, nil
}

func (c credsFromConn) Clone() credentials.TransportCredentials {
	return c
}

func (c credsFromConn) OverrideServerName(_ string) error {
	return errors.New("not supported")
}
