package reportcheck

import (
	"fmt"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"google.golang.org/protobuf/proto"
)

const (
	// Observed on RHEL 9/10 VMs (2026-06): ~524 packages, ~449 KiB per report.
	warnMinPackages = 5
	warnMaxBytes    = 2 << 20 // 2 MiB (~3× observed max of ~512 KiB)
)

// IsViable returns whether the report is safe to forward, plus a
// human-readable warning (empty if clean). The caller is responsible
// for logging and adding VM context.
func IsViable(report *v4.IndexReport) (bool, string) {
	if report == nil {
		return false, "nil report"
	}

	pkgs := len(report.GetContents().GetPackages())
	size := proto.Size(report)

	if pkgs == 0 {
		return true, fmt.Sprintf("zero packages (state=%s, size=%d bytes) — VM may have no package manager or scan failed silently",
			report.GetState(), size)
	}
	if pkgs < warnMinPackages {
		return true, fmt.Sprintf("only %d packages — unexpectedly low for a production VM", pkgs)
	}
	if size > warnMaxBytes {
		return true, fmt.Sprintf("report is %d bytes (%d packages) — unusually large", size, pkgs)
	}

	return true, ""
}
