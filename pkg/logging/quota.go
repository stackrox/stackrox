package logging

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	defaultMaxLogLineQuotaPerInterval = 100
	defaultLogLineQuotaIntervalSecs   = 10
)

var (
	maxLogLineQuotaPerInterval, logLineQuotaIntervalSecs int64 = func() (quotaPerInterval int64, quotaIntervalSecs int64) {
		quotaPerInterval, quotaIntervalSecs = defaultMaxLogLineQuotaPerInterval, defaultLogLineQuotaIntervalSecs

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
			quotaPerInterval = maxLines
		}
		if intervalSecs != 0 {
			quotaIntervalSecs = intervalSecs
		}
		return
	}()
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
