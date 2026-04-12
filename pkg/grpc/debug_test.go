package grpc

import (
	"bufio"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

func newDebugLogger(t *testing.T) *debugLoggerImpl {
	return &debugLoggerImpl{
		log: t,
	}
}

func testPrintSocketInfo(t *testing.T, ports ...uint64) error {
	errList := errorhelpers.NewErrorList("print socket info")
	for _, fName := range procFiles {
		if err := testPrintSocketInfoFromProcFile(t, fName, ports...); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}

func testPrintSocketInfoFromProcFile(t *testing.T, fName string, ports ...uint64) (err error) {
	shouldPrintPort := func(port uint64, ports ...uint64) bool {
		for _, p := range ports {
			if p == port {
				return true
			}
		}
		return false
	}
	getStateString := func(code uint64) string {
		codeToState := map[uint64]string{
			0x01: "ESTABLISHED",
			0x02: "SYN_SENT",
			0x03: "SYN_RECV",
			0x04: "FIN_WAIT1",
			0x05: "FIN_WAIT2",
			0x06: "TIME_WAIT",
			0x07: "CLOSE",
			0x08: "CLOSE_WAIT",
			0x09: "LAST_ACK",
			0x0a: "LISTEN",
			0x0b: "CLOSING",
		}
		str, found := codeToState[code]
		if !found {
			return "UNKNOWN"
		}
		return str
	}
	f, openErr := os.Open(fName)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = closeErr
		}
	}()
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	for scanner.Scan() {
		columns := strings.Fields(scanner.Text())
		if len(columns) < 12 {
			return errors.Errorf("not enough columns in the line: %q", scanner.Text())
		}
		fields := strings.Split(columns[1], ":")
		if len(fields) < 2 {
			return errors.Errorf("not enouch fields in the address column: %q", columns[1])
		}
		port, parseErr := strconv.ParseUint(fields[1], 16, 16)
		if parseErr != nil {
			return parseErr
		}
		if !shouldPrintPort(port, ports...) {
			continue
		}
		code, parseErr := strconv.ParseUint(columns[3], 16, 8)
		if parseErr != nil {
			return parseErr
		}
		t.Logf("Port %d is in %q state", port, getStateString(code))
	}
	return err
}

func testPrintStackTraceInfo(_ *testing.T) error {
	errList := errorhelpers.NewErrorList("print stacktrace info")
	for _, p := range pprof.Profiles() {
		if err := p.WriteTo(os.Stderr, 2); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}
