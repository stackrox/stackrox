package utils

import (
	"os"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

const auditFile = "/etc/audit/audit.rules"

// CheckAudit checks to see if a file is currently tracked in auditd
func CheckAudit(file string) (result v1.CheckResult) {
	// If the file doesn't exist then it will throw an error
	if _, err := os.Stat(file); err != nil {
		Info(&result)
		AddNotef(&result, "File or Directory %v does not exist", file)
		return
	}
	auditFile, err := ReadFile(auditFile)
	if err != nil {
		Warn(&result)
		AddNotes(&result, "Error reading %v: err.Error()", auditFile)
		return
	}
	if !strings.Contains(auditFile, file) {
		Warn(&result)
		AddNotef(&result, "Audit file /etc/audit/audit.rules does not contain %v", file)
		return
	}
	Pass(&result)
	return
}
