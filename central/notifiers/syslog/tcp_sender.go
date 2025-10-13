package syslog

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	initialBackoff             = 1 * time.Second
	maxBackoff                 = 5 * time.Minute
	backoffRandomizationFactor = .8
)

var (
	// Dial timeout.  Otherwise we'll wait TCPs default 30 second timeout for things like waiting for a TLS handshake.
	timeout = env.SyslogUploadTimeout.DurationSetting()
)

type connWrapper struct {
	conn   net.Conn
	failed concurrency.Signal
}

func (c *connWrapper) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

type tcpSender struct {
	connPtr   atomic.Value
	stop      concurrency.Signal
	available concurrency.Signal

	fullHostname  string
	useTLS        bool
	skipTLSVerify bool
}

func getTCPSender(endpointConfig *storage.Syslog_TCPConfig) (syslogSender, error) {
	fullHostname, err := validateRemoteConfig(endpointConfig)
	if err != nil {
		return nil, err
	}

	sender := &tcpSender{
		stop:          concurrency.NewSignal(),
		available:     concurrency.NewSignal(),
		fullHostname:  fullHostname,
		useTLS:        endpointConfig.GetUseTls(),
		skipTLSVerify: endpointConfig.GetSkipTlsVerify(),
	}

	// the failed signal is uninitialized so it will be triggered from the start.
	sender.connPtr.Store(&connWrapper{})
	// Start the reconnect goroutine to set up the initial connection and monitor for connection failures.
	go sender.reconnect()

	return sender, nil
}

func validateRemoteConfig(endpointConfig *storage.Syslog_TCPConfig) (string, error) {
	if endpointConfig == nil {
		return "", errors.New("no TCP syslog endpoint config found")
	}

	if endpointConfig.GetHostname() == "" {
		return "", errors.New("no host name in endpoint config")
	}

	port := endpointConfig.GetPort()
	if port < 1 || port > 65535 {
		return "", errors.Errorf("invalid port number %d must be between 1 and 65535", port)
	}

	hostName := fmt.Sprintf("%s:%d", endpointConfig.GetHostname(), endpointConfig.GetPort())

	sysURL := fmt.Sprintf("tcp://%s", hostName)

	_, err := url.ParseRequestURI(sysURL)

	if err != nil {
		return "", errors.New("invalid host name")
	}

	return hostName, nil
}

func (s *tcpSender) dialWithRetry() (net.Conn, error) {
	// Get a non-tls dialFunc
	tcpDialFunc := proxy.AwareDialContext
	// If we're using TLS upgrade to a TLS dialFunc
	if s.useTLS {
		tlsConfig := &tls.Config{InsecureSkipVerify: s.skipTLSVerify}
		tcpDialFunc = func(ctx context.Context, addr string) (net.Conn, error) {
			return proxy.AwareDialContextTLS(ctx, addr, tlsConfig)
		}
	}

	// Create a retryable dial func, returning a permanent error if the stop signal has signaled.
	ctx := concurrency.AsContext(&s.stop)
	var conn net.Conn
	dial := func() error {
		var err error
		dialCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		conn, err = tcpDialFunc(dialCtx, s.fullHostname)
		return err
	}

	// Configure a retry object and start retrying the TCP connection.
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = maxBackoff
	eb.InitialInterval = initialBackoff
	eb.RandomizationFactor = backoffRandomizationFactor
	// backoff.WithContext will return a permanent error if the context has expired.
	err := backoff.Retry(dial, backoff.WithContext(eb, ctx))

	return conn, err
}

func (s *tcpSender) reconnect() {
	for {
		curConn := s.connPtr.Load().(*connWrapper)
		// If the stop signal has been activated we don't want to try to reconnect.
		if s.stop.IsDone() {
			curConn.Close()
			return
		}

		select {
		case <-curConn.failed.WaitC():
			// Senders should wait for the new connection
			s.available.Reset()

			// Close old connection
			curConn.Close()

			// Store an empty connWrapper.  The fail signal will be set and the connection will be nil.  A set fail
			// signal will cause us to re-enter this loop if we break out somehow without establishing a new connection
			// and a nil connection will allow senders to fail fast while we reconnect.
			s.connPtr.Store(&connWrapper{})

			// Create new connection.  Will return null if the stop signal is activated.
			conn, err := s.dialWithRetry()
			if err != nil {
				continue
			}

			newConn := &connWrapper{
				conn:   conn,
				failed: concurrency.NewSignal(),
			}
			s.connPtr.Store(newConn)
			s.available.Signal()
		case <-s.stop.WaitC():
			curConn.Close()
			return
		}
	}
}

func (s *tcpSender) SendSyslog(syslogBytes []byte) error {
	// Don't try to send before we've initialized the connection
	select {
	case <-s.available.WaitC():
	case <-s.stop.WaitC():
		return errors.New("syslog notifier stopped")
	case <-time.After(timeout):
		return errors.New("timed out waiting for a syslog connection")
	}

	conn := s.connPtr.Load().(*connWrapper)
	// conn.conn can be nil if we are currently reconnecting.  Fail fast.
	if conn.conn == nil {
		return errors.New("no TCP connection to syslog receiver")
	}

	// Prepend "NUMBER_OF_BYTES_AS_A_STRING " to the syslog header as per the RFC 5424 spec for sending syslog over TCP
	byteLen := len(syslogBytes)
	syslogFrame := append([]byte(strconv.Itoa(byteLen)), byte(' '))
	syslogFrame = append(syslogFrame, syslogBytes...)

	// Try to send.  If we receive an error, signal a reconnect.
	_, err := conn.conn.Write(syslogFrame)
	if err != nil {
		conn.failed.Signal()
		log.Errorw("Failed to write to Syslog", logging.Err(err))
	}
	return err
}

func (s *tcpSender) Cleanup() {
	s.stop.Signal()
}
