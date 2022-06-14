package docker

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/compliance/msgfmt"
)

var (
	dockerDirRegex = regexp.MustCompile(`/var/lib/docker\s`)
)

const auditFile = "/etc/audit/audit.rules"

func init() {
	standards.RegisterChecksForStandard(standards.CISDocker, map[string]*standards.CheckAndMetadata{
		standards.CISDockerCheckName("1_1_1"): common.NoteCheck("Ensure the container host has been Hardened"),
		standards.CISDockerCheckName("1_1_2"): common.NoteCheck(" Ensure that the version of Docker is up to date"),
		standards.CISDockerCheckName("1_2_1"): {
			CheckFunc: containerPartition,
		},
		standards.CISDockerCheckName("1_2_2"):  common.NoteCheck("Ensure only trusted users are allowed to control Docker daemon"),
		standards.CISDockerCheckName("1_2_3"):  auditCheckFunc("/usr/bin/dockerd", "/usr/bin/dockerd-current"),
		standards.CISDockerCheckName("1_2_4"):  auditCheckFunc("/var/lib/docker"),
		standards.CISDockerCheckName("1_2_5"):  auditCheckFunc("/etc/docker"),
		standards.CISDockerCheckName("1_2_6"):  auditCheckFunc("docker.service"),
		standards.CISDockerCheckName("1_2_7"):  auditCheckFunc("docker.socket"),
		standards.CISDockerCheckName("1_2_8"):  auditCheckFunc("/etc/default/docker"),
		standards.CISDockerCheckName("1_2_9"):  auditCheckFunc("/etc/sysconfig/docker"),
		standards.CISDockerCheckName("1_2_10"): auditCheckFunc("/etc/docker/daemon.json"),
		standards.CISDockerCheckName("1_2_11"): auditCheckFunc("/usr/bin/containerd", "/usr/bin/docker-containerd"),
		standards.CISDockerCheckName("1_2_12"): auditCheckFunc("/usr/sbin/runc", "/usr/bin/runc"),

		standards.CISDockerCheckName("5_22"): {
			CheckFunc: privilegedDockerExec,
		},
		standards.CISDockerCheckName("5_23"): {
			CheckFunc: userDockerExec,
		},
	})
}

func auditCheckFunc(files ...string) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			f, ok := complianceData.Files[auditFile]
			if !ok {
				return common.FailListf("Audit file %q does not exist", auditFile)
			}
			var found bool
			var results []*storage.ComplianceResultValue_Evidence
			for _, file := range files {
				fileByte := []byte(file)
				// systemd services will not show up in files, but they do exist
				if _, ok := complianceData.Files[file]; !ok && filepath.Ext(file) != ".service" {
					continue
				}
				found = true
				if !bytes.Contains(f.GetContent(), fileByte) {
					results = append(results, common.Failf("Audit file %q does not contain file %q", auditFile, file))
					continue
				}
				results = append(results, common.Passf("Audit file %q contains file %q", auditFile, file))
			}
			if !found {
				for _, f := range files {
					results = append(results, common.Passf("File %q does not exist so it does not need to be audited", f))
				}
			}
			return results
		},
		Metadata: &standards.Metadata{
			InterpretationText: fmt.Sprintf("StackRox checks that auditd rules exist for files %s (if present)", msgfmt.FormatStrings(files...)),
			TargetKind:         framework.NodeKind,
		},
	}
}

const procMountsPath = "/proc/mounts"

func containerPartition(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
	f, ok := complianceData.Files[procMountsPath]
	if !ok {
		return common.FailListf("File %q does not exist", procMountsPath)
	}
	if !dockerDirRegex.Match(f.GetContent()) {
		return common.FailListf("File %q does not contain file /var/lib/docker", procMountsPath)
	}
	return common.PassListf("File %q contains file %q", procMountsPath, "/var/lib/docker")
}

const auditLogFile = "/var/log/audit/audit.log"

func privilegedDockerExec(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
	execByte := []byte("exec")
	privilegedByte := []byte("privileged")
	f, ok := complianceData.Files[auditLogFile]
	if !ok {
		return common.FailListf("Audit log file %q does not exist", auditLogFile)
	}
	if bytes.Contains(f.GetContent(), execByte) && bytes.Contains(f.GetContent(), privilegedByte) {
		return common.FailListf("Docker exec was used with the --privileged flag: %s", f.GetContent())
	}
	return common.PassList("No Docker execs found with --privileged flag")
}

func userDockerExec(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
	execByte := []byte("exec")
	userByte := []byte("user=root")
	f, ok := complianceData.Files[auditLogFile]
	if !ok {
		return common.FailListf("Audit log file %q does not exist", auditLogFile)
	}
	if bytes.Contains(f.GetContent(), execByte) && bytes.Contains(f.GetContent(), userByte) {
		return common.FailListf("Docker exec was used with the --user flag: %s", f.GetContent())
	}
	return common.PassList("No Docker execs found with --user flag")
}
