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
	signal.ExecFilePath = sanitizeString(signal.GetExecFilePath())
	signal.Name = sanitizeString(signal.GetName())
	signal.Args = sanitizeString(signal.GetArgs())
	for _, lineage := range signal.GetLineageInfo() {
		lineage.ParentExecFilePath = sanitizeString(lineage.GetParentExecFilePath())
	}
}

// NetworkEndpoint ensures that that endpoint will comply with UTF8 encoding
func NetworkEndpoint(endpoint *sensor.NetworkEndpoint) {
	originator := endpoint.GetOriginator()
	if originator == nil {
		return
	}
	originator.ProcessExecFilePath = sanitizeString(originator.GetProcessExecFilePath())
	originator.ProcessName = sanitizeString(originator.GetProcessName())
	originator.ProcessArgs = sanitizeString(originator.GetProcessArgs())
}
