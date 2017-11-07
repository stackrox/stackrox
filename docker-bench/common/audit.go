package common

import (
	"os"
	"strings"
)

const auditFile = "/etc/audit/audit.rules"

// CheckAudit checks to see if a file is currently tracked in auditd
func CheckAudit(file string) (result TestResult) {
	// If the file doesn't exist then it will throw an error
	if _, err := os.Stat(file); err != nil {
		result.Info()
		result.AddNotef("File or Directory %v does not exist", file)
		return
	}
	auditFile, err := ReadFile(auditFile)
	if err != nil {
		result.Warn()
		result.AddNotes("Error reading %v: err.Error()", auditFile)
		return
	}
	if !strings.Contains(auditFile, file) {
		result.Warn()
		result.AddNotef("Audit file /etc/audit/audit.rules does not contain %v", file)
		return
	}
	result.Pass()
	return
}
