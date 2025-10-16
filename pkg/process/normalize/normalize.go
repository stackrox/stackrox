package normalize

import (
	"strings"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
)

func sanitizeString(s string) string {
	s = strings.ToValidUTF8(s, "")
	return strings.Trim(s, "\x00")
}

// Indicator ensures that that indicator will comply with UTF8 encoding
func Indicator(indicator *storage.ProcessIndicator) {
	signal := indicator.GetSignal()
	if signal == nil {
		return
	}
	signal.SetExecFilePath(sanitizeString(signal.GetExecFilePath()))
	signal.SetName(sanitizeString(signal.GetName()))
	signal.SetArgs(sanitizeString(signal.GetArgs()))
	for _, lineage := range signal.GetLineageInfo() {
		lineage.SetParentExecFilePath(sanitizeString(lineage.GetParentExecFilePath()))
	}
}

// NetworkEndpoint ensures that that endpoint will comply with UTF8 encoding
func NetworkEndpoint(endpoint *sensor.NetworkEndpoint) {
	originator := endpoint.GetOriginator()
	if originator == nil {
		return
	}
	originator.SetProcessExecFilePath(sanitizeString(originator.GetProcessExecFilePath()))
	originator.SetProcessName(sanitizeString(originator.GetProcessName()))
	originator.SetProcessArgs(sanitizeString(originator.GetProcessArgs()))
}
