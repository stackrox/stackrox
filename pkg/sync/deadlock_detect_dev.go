//go:build !release && !go1.17

package sync

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

const (
	// deadlockCheckInterval specifies the interval in which deadlock checks are performed. Since any case in which
	// `[R]Unlock()` is called eventually (but after the timeout) will be caught by the respective check in
	// `[R]Unlock()` itself, this doesn't need to run all that frequently.
	deadlockCheckInterval = 2 * time.Minute
)

var (
	// panicOnTimeoutRegex finds invocations of panicOnTimeoutMarked on the stack, which allow figuring out since
	// when an action is waiting to complete.
	panicOnTimeoutRegex = regexp.MustCompile(`\ngithub\.com/stackrox/rox/pkg/sync\.panicOnTimeoutMarked\([^\n]*, 0x([0-9a-f]+), 0x([0-9a-f]+)\)\n`)
)

func init() {
	go detectDeadlock()
}

// allStackTraces returns a byte slice with stacktraces of all goroutines, using the given slice as a buffer hint.
func allStackTraces(buf []byte) []byte {
	if cap(buf) == 0 {
		buf = make([]byte, 4096)
	} else {
		buf = buf[:cap(buf)]
	}

	written := runtime.Stack(buf, true)
	for written >= len(buf) {
		buf = make([]byte, 2*len(buf))
		written = runtime.Stack(buf, true)
	}
	return buf[:written]
}

// checkStacktraceMatch checks the given stacktrace of panicOnTimeoutRegex on buf, which contains the stack
// traces of all goroutines.
func checkStacktraceMatch(buf []byte, subMatches []int) {
	if len(subMatches) < 6 {
		return
	}
	timeoutVal, err := strconv.ParseInt(string(buf[subMatches[2]:subMatches[3]]), 16, 64)
	if err != nil {
		return
	}
	timeout := time.Duration(timeoutVal)
	if timeout <= 0 {
		return
	}

	tsVal, err := strconv.ParseInt(string(buf[subMatches[4]:subMatches[5]]), 16, 64)
	if err != nil {
		return
	}
	if time.Since(time.Unix(0, tsVal)) <= timeout {
		return
	}

	// Find the start and end of the relevant Goroutine's stack trace. Goroutines are delimited by empty lines.
	stackTraceStart := bytes.LastIndex(buf[:subMatches[0]], []byte("\n\n"))
	if stackTraceStart == -1 {
		stackTraceStart = 0
	} else {
		stackTraceStart += 2 // account for "\n\n"
	}
	stackTraceEnd := bytes.Index(buf[subMatches[1]:], []byte("\n\n"))
	if stackTraceEnd == -1 {
		stackTraceEnd = len(buf)
	} else {
		stackTraceEnd += subMatches[1] // account for start index
	}

	stackTrace := buf[stackTraceStart:stackTraceEnd]
	_, _ = fmt.Fprintf(os.Stderr, "Some action took more than %v to complete. Stack trace:\n%s\n", timeout, stackTrace)
	kill()
}

func detectDeadlock() {
	var buf []byte

	for range time.NewTicker(deadlockCheckInterval).C {
		buf = allStackTraces(buf)
		matches := panicOnTimeoutRegex.FindAllSubmatchIndex(buf, -1)
		for _, subMatches := range matches {
			checkStacktraceMatch(buf, subMatches)
		}
	}
}
