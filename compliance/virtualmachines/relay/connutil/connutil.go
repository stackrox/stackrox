// Package connutil contains helpers for bounded reads on generic net.Conn
// implementations.
package connutil

import (
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// ReadFromConn reads from conn with a deadline and rejects payloads above
// maxSize bytes.
func ReadFromConn(conn net.Conn, maxSize int, timeout time.Duration) ([]byte, error) {
	log.Debugf("Reading from connection (max bytes: %d, timeout: %s)", maxSize, timeout)

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, errors.Wrap(err, "setting read deadline on connection")
	}

	// Even if not strictly required, we limit the amount of data to be read to protect Sensor against large workloads.
	// Add 1 to the limit so we can detect oversized data. If we used exactly maxSize, we couldn't tell the difference
	// between a valid message of exactly maxSize bytes and an invalid message that's larger than maxSize (both would
	// read maxSize bytes). With maxSize+1, reading more than maxSize bytes means the original data was too large.
	limitedReader := io.LimitReader(conn, int64(maxSize+1))
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, errors.Wrapf(err, "reading data from connection (remote: %v)", conn.RemoteAddr())
	}

	if len(data) > maxSize {
		return nil, errors.Errorf("data size exceeds the limit (%d bytes, remote: %v)", maxSize, conn.RemoteAddr())
	}

	return data, nil
}
