package k8sintrospect

import (
	"bytes"
)

// truncateLogData truncates log file data to the given maximum size, making a best-effort approach to preserve whole
// lines only.
func truncateLogData(logData []byte, maxLogFileSize, maxFirstLineCutOff int) []byte {
	if len(logData) <= maxLogFileSize {
		return logData
	}
	// In the following array index expression, subtract 1 to transparently handle the line-boundary case when
	// truncating.
	logData = logData[len(logData)-maxLogFileSize-1:]
	firstNewline := bytes.IndexByte(logData, '\n')
	if firstNewline == -1 || firstNewline > maxFirstLineCutOff {
		firstNewline = 0
	}
	logData = logData[firstNewline+1:]
	return logData
}
