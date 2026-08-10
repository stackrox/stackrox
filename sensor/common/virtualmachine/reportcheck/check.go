package reportcheck

import (
	"fmt"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"google.golang.org/protobuf/proto"
)

// IsViable returns whether the report is safe to forward, plus a
// human-readable warning (empty if clean). The caller is responsible
// for logging and adding VM context.
//
// warnMaxBytes is the size, in bytes, above which a report is flagged as
// unusually large. Callers derive it from the configured pull-mode
// response-size ceiling (see env.VirtualMachinesPullMaxResponseSizeKB)
// rather than a value hardcoded here, so the warning stays in sync with
// that ceiling instead of drifting from it.
func IsViable(report *v4.IndexReport, warnMaxBytes int) (bool, string) {
	if report == nil {
		return false, "nil report"
	}

	pkgs := len(report.GetContents().GetPackages())
	size := proto.Size(report)

	if pkgs == 0 {
		return true, fmt.Sprintf("zero packages (state=%s, size=%d bytes) — VM may have no package manager or scan failed silently",
			report.GetState(), size)
	}
	// Debuggability: the protocol between Sensor and roxagent has a limit (currently 16MB).
	// If the report is larger than the limit, the transmission will fail.
	// We want to have a trace in the logs to help debug the issue in case it happens.
	if size > warnMaxBytes {
		return true, fmt.Sprintf("report is %d bytes (%d packages)", size, pkgs)
	}

	return true, ""
}
