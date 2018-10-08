package utils

import (
	"os"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

const auditFile = "/etc/audit/audit.rules"

// CheckAudit Checks to see if a file is currently tracked in auditd
func CheckAudit(file string) (result v1.CheckResult) {
	// If the file doesn't exist then it will throw an error
	if _, err := os.Stat(ContainerPath(file)); err != nil {
		Info(&result)
		AddNotef(&result, "File or Directory %v does not exist", file)
		return
	}

	auditFileData, err := ReadFile(ContainerPath(auditFile))
	if err != nil {
		Warn(&result)
		AddNotef(&result, "Error reading %s: %s", auditFile, err.Error())
		return
	}
	if !strings.Contains(auditFileData, file) {
		Warn(&result)
		AddNotef(&result, "Audit file %v does not contain %v", auditFile, file)
		return
	}
	Pass(&result)
	return
}
