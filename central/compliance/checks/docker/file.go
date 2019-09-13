package docker

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

func init() {
	framework.MustRegisterChecks(
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:1_1_1", "Ensure the container host has been Hardened"),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:1_1_2", " Ensure that the version of Docker is up to date"),
		framework.NewCheckFromFunc(framework.CheckMetadata{ID: "CIS_Docker_v1_2_0:1_2_1", Scope: framework.NodeKind}, containerPartition),
		common.PerNodeNoteCheck("CIS_Docker_v1_2_0:1_2_2", "Ensure only trusted users are allowed to control Docker daemon"),
		auditCheck("CIS_Docker_v1_2_0:1_2_3", "/usr/bin/dockerd"),
		auditCheck("CIS_Docker_v1_2_0:1_2_4", "/var/lib/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_5", "/etc/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_6", "docker.service"),
		auditCheck("CIS_Docker_v1_2_0:1_2_7", "docker.socket"),
		auditCheck("CIS_Docker_v1_2_0:1_2_8", "/etc/default/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_9", "/etc/sysconfig/docker"),
		auditCheck("CIS_Docker_v1_2_0:1_2_10", "/etc/docker/daemon.json"),
		auditCheck("CIS_Docker_v1_2_0:1_2_11", "/usr/bin/containerd"),
		auditCheck("CIS_Docker_v1_2_0:1_2_12", "/usr/sbin/runc"),

		framework.NewCheckFromFunc(framework.CheckMetadata{ID: "CIS_Docker_v1_2_0:5_22", Scope: framework.NodeKind}, privilegedDockerExec),
		framework.NewCheckFromFunc(framework.CheckMetadata{ID: "CIS_Docker_v1_2_0:5_23", Scope: framework.NodeKind}, userDockerExec),
	)
}

const auditFile = "/etc/audit/audit.rules"

func auditCheck(name, file string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 name,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("StackRox checks that auditd rules exist for file %s (if present)", file),
	}
	return framework.NewCheckFromFunc(md, auditCheckFunc(file))
}

func auditCheckFunc(file string) framework.CheckFunc {
	fileByte := []byte(file)
	return common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[auditFile]
		if !ok {
			framework.FailNowf(ctx, "Audit file %q does not exist", auditFile)
		}
		if _, ok := returnData.Files[file]; !ok {
			framework.PassNowf(ctx, "File %q does not exist so it does not need to be audited", file)
		}
		if !bytes.Contains(f.GetContent(), fileByte) {
			framework.FailNowf(ctx, "Audit file %q does not contain file %q", auditFile, file)
		}
		framework.PassNowf(ctx, "Audit file %q contains file %q", auditFile, file)
	})
}

const procMountsPath = "/proc/mounts"

func containerPartition(funcCtx framework.ComplianceContext) {
	r := regexp.MustCompile(`/var/lib/docker\s`)
	common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[procMountsPath]
		if !ok {
			framework.FailNowf(ctx, "File %q does not exist", procMountsPath)
		}
		if !r.Match(f.GetContent()) {
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
	userByte := []byte("user")
	privilegedByte := []byte("privileged")
	common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[auditLogFile]
		if !ok {
			framework.FailNowf(ctx, "Audit log file %q does not exist", auditLogFile)
		}
		if bytes.Contains(f.GetContent(), userByte) && bytes.Contains(f.GetContent(), privilegedByte) {
			framework.FailNowf(ctx, "Docker exec was used with the --user flag: %s", f.GetContent())
		}
		framework.Pass(ctx, "No Docker execs found with --user flag")
	})(funcCtx)
}
