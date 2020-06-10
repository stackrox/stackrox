package logging

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultMaxLogLineQuotaPerInterval = 100
	defaultLogLineQuotaIntervalSecs   = 10
)

var (
	numLogLines   int64                     // number of log lines that were attempted to be written in the current window
	windowStartTS       = time.Now().Unix() // timestamp (unix seconds) of the window when the first log line was written

	maxLogLineQuotaPerInterval int64 = defaultMaxLogLineQuotaPerInterval
	logLineQuotaIntervalSecs   int64 = defaultLogLineQuotaIntervalSecs
)

func parseLogLineQuotaSetting(setting string) (maxLines, intervalSecs int64, err error) {
	parts := strings.SplitN(setting, "/", 2)
	if len(parts) == 0 {
		return
	}

	maxLinesStr := strings.TrimSpace(parts[0])
	if maxLinesStr != "" {
		maxLines, err = strconv.ParseInt(maxLinesStr, 10, 64)
		if err != nil {
			return 0, 0, errors.Wrap(err, "parsing first component")
		} else if maxLines <= 0 {
			return 0, 0, errors.Errorf("maximum number of log lines per interval must be positive (is: %d)", maxLines)
		}
	}

	if len(parts) < 2 {
		return
	}

	intervalSecsStr := strings.TrimSpace(parts[1])
	if intervalSecsStr == "" {
		return
	}

	intervalSecs, err = strconv.ParseInt(intervalSecsStr, 10, 64)
	if err != nil {
		return 0, 0, errors.Wrap(err, "parsing second component")
	} else if intervalSecs <= 0 {
		return 0, 0, errors.Errorf("log line quota interval must be positive (is: %d)", maxLines)
	}

	return
}

func init() {
	quotaStr := os.Getenv("MAX_LOG_LINE_QUOTA")
	if quotaStr == "" {
		return
	}

	maxLines, intervalSecs, err := parseLogLineQuotaSetting(quotaStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse MAX_LOG_LINE_QUOTA setting %q: %v\n", quotaStr, err)
		return
	}

	if maxLines != 0 {
		maxLogLineQuotaPerInterval = maxLines
	}
	if logLineQuotaIntervalSecs != 0 {
		logLineQuotaIntervalSecs = intervalSecs
	}
}

func checkLogLineQuota(output *log.Logger) bool {
	if atomic.AddInt64(&numLogLines, 1) <= maxLogLineQuotaPerInterval {
		return true
	}
	startTS := atomic.LoadInt64(&windowStartTS)
	nowTS := time.Now().Unix()
	if nowTS-startTS < logLineQuotaIntervalSecs {
		return false
	}

	// Make sure only a single goroutine gets to reset the window.
	// Note: a side effect is that sometimes log messages might get throttled when they
	// shouldn't, e.g., when the counter is exceeded but the window is no longer current, and
	// log calls in two concurrent goroutines determine that to be the case. However,
	// this seems a fairly small price to pay for the protection we are getting at almost no cost.
	if !atomic.CompareAndSwapInt64(&windowStartTS, startTS, nowTS) {
		return false
	}

	oldNumLogLines := atomic.SwapInt64(&numLogLines, 1)
	numThrottled := oldNumLogLines - maxLogLineQuotaPerInterval - 1
	if numThrottled > 0 {
		fmt.Fprintln(output.Writer(), "[OMITTED APPROX.", numThrottled, "LOG LINES DUE TO THROTTLING]")
	}
	return true
}
