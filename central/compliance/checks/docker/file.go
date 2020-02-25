package docker

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/msgfmt"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

var (
	dockerDirRegex = regexp.MustCompile(`/var/lib/docker\s`)
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:1_1_1", "Ensure the container host has been Hardened"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:1_1_2", " Ensure that the version of Docker is up to date"),
		framework.NewCheckFromFunc(framework.CheckMetadata{ID: "CIS_Docker_v1_2_0:1_2_1", Scope: framework.NodeKind}, containerPartition),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:1_2_2", "Ensure only trusted users are allowed to control Docker daemon"),
		auditCheck("CIS_Docker_v1_2_0:1_2_3", "/usr/bin/dockerd", "/usr/bin/dockerd-current"),
		auditCheck("CIS_Docker_v1_2_0:1_2_4", "/var/lib/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_5", "/etc/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_6", "docker.service"),
		auditCheck("CIS_Docker_v1_2_0:1_2_7", "docker.socket"),
		auditCheck("CIS_Docker_v1_2_0:1_2_8", "/etc/default/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_9", "/etc/sysconfig/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_10", "/etc/docker/daemon.json"),
		auditCheck("CIS_Docker_v1_2_0:1_2_11", "/usr/bin/containerd", "/usr/bin/docker-containerd"),
		auditCheck("CIS_Docker_v1_2_0:1_2_12", "/usr/sbin/runc", "/usr/bin/runc"),

		framework.NewCheckFromFunc(framework.CheckMetadata{ID: "CIS_Docker_v1_2_0:5_22", Scope: framework.NodeKind}, privilegedDockerExec),
		framework.NewCheckFromFunc(framework.CheckMetadata{ID: "CIS_Docker_v1_2_0:5_23", Scope: framework.NodeKind}, userDockerExec),
	)
}

const auditFile = "/etc/audit/audit.rules"

func auditCheck(name string, files ...string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that auditd rules exist for files %s (if present)", msgfmt.FormatStrings(files...)),
	}
	return framework.NewCheckFromFunc(md, auditCheckFunc(files...))
}

func auditCheckFunc(files ...string) framework.CheckFunc {
	return common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[auditFile]
		if !ok {
			framework.FailNowf(ctx, "Audit file %q does not exist", auditFile)
		}
		var found bool
		for _, file := range files {
			fileByte := []byte(file)
			// systemd services will not show up in files, but they do exist
			if _, ok := returnData.Files[file]; !ok && filepath.Ext(file) != ".service" {
				continue
			}
			found = true
			if !bytes.Contains(f.GetContent(), fileByte) {
				framework.Failf(ctx, "Audit file %q does not contain file %q", auditFile, file)
				continue
			}
			framework.Passf(ctx, "Audit file %q contains file %q", auditFile, file)
		}
		if !found {
			for _, f := range files {
				framework.Passf(ctx, "File %q does not exist so it does not need to be audited", f)
			}
		}
	})
}

const procMountsPath = "/proc/mounts"

func containerPartition(funcCtx framework.ComplianceContext) {
	common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[procMountsPath]
		if !ok {
			framework.FailNowf(ctx, "File %q does not exist", procMountsPath)
		}
		if !dockerDirRegex.Match(f.GetContent()) {
			framework.FailNowf(ctx, "File %q does not contain file /var/lib/docker", procMountsPath)
		}
		framework.PassNowf(ctx, "File %q contains file %q", procMountsPath, "/var/lib/docker")
	})(funcCtx)
}

const auditLogFile = "/var/log/audit/audit.log"

func privilegedDockerExec(funcCtx framework.ComplianceContext) {
	execByte := []byte("exec")
	privilegedByte := []byte("privileged")
	common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[auditLogFile]
		if !ok {
			framework.FailNowf(ctx, "Audit log file %q does not exist", auditLogFile)
		}
		if bytes.Contains(f.GetContent(), execByte) && bytes.Contains(f.GetContent(), privilegedByte) {
			framework.FailNowf(ctx, "Docker exec was used with the --privileged flag: %s", f.GetContent())
		}
		framework.Pass(ctx, "No Docker execs found with --privileged flag")
	})(funcCtx)
}

func userDockerExec(funcCtx framework.ComplianceContext) {
	execByte := []byte("exec")
	userByte := []byte("user=root")
	common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[auditLogFile]
		if !ok {
			framework.FailNowf(ctx, "Audit log file %q does not exist", auditLogFile)
		}
		if bytes.Contains(f.GetContent(), execByte) && bytes.Contains(f.GetContent(), userByte) {
			framework.FailNowf(ctx, "Docker exec was used with the --user flag: %s", f.GetContent())
		}
		framework.Pass(ctx, "No Docker execs found with --user flag")
	})(funcCtx)
}
