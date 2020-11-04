package syslog

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// Dial timeout.  Otherwise we'll wait TCPs default 30 second timeout for things like waiting for a TLS handshake.
	timeout = 5 * time.Second
)

type tcpSender struct {
	conn           net.Conn
	endpointConfig *storage.Syslog_TCPConfig
}

func getTCPSender(endpointConfig *storage.Syslog_TCPConfig) (syslogSender, error) {
	conn, err := validateRemoteConfig(endpointConfig)
	if err != nil {
		return nil, err
	}

	return &tcpSender{
		conn:           conn,
		endpointConfig: endpointConfig,
	}, nil
}

func validateRemoteConfig(endpointConfig *storage.Syslog_TCPConfig) (net.Conn, error) {
	if endpointConfig == nil {
		return nil, errors.New("no TCP syslog endpoint config found")
	}

	if endpointConfig.GetHostname() == "" {
		return nil, errors.New("no host name in endpoint config")
	}

	port := endpointConfig.GetPort()
	if port < 1 || port > 65353 {
		return nil, errors.Errorf("invalid port number %d must be between 1 and 65353", port)
	}

	fullHostname := fmt.Sprintf("%s:%d", endpointConfig.GetHostname(), endpointConfig.GetPort())

	// If we aren't using TLS don't try TLS
	if !endpointConfig.GetUseTls() {
		return net.DialTimeout("tcp", fullHostname, timeout)
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: endpointConfig.GetSkipTlsVerify()}
	// tls.Dial uses the default dialer.  The only difference here should be the added timeout.
	var dialerWithTimeout net.Dialer
	dialerWithTimeout.Timeout = timeout
	return tls.DialWithDialer(&dialerWithTimeout, "tcp", fullHostname, tlsConfig)
}

func (s *tcpSender) SendSyslog(syslogBytes []byte) error {
	byteLen := len(syslogBytes)
	// Prepend "NUMBER_OF_BYTES_AS_A_STRING " to the syslog header as per the RFC 5424 spec for sending syslog over TCP
	byteLenHeader := append([]byte(strconv.Itoa(byteLen)), []byte(" ")...)
	syslogBytes = append(byteLenHeader, syslogBytes...)

	_, err := s.conn.Write(syslogBytes)
	return err
}

func (s *tcpSender) Cleanup() error {
	return s.conn.Close()
}
