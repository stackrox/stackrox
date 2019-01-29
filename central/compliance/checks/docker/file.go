package docker

import (
	"bytes"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

func init() {
	framework.MustRegisterChecks(
		framework.NewCheckFromFunc("CIS_Docker_v1_1_0:1_1", framework.NodeKind, nil, containerPartition),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:1_2", "Ensure the host is hardened with the latest kernel"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:1_3", "Ensure that Docker is updated"),
		common.PerNodeNoteCheck("CIS_Docker_v1_1_0:1_4", "Ensure that only trusted users can access the Docker daemon"),
		auditCheck("CIS_Docker_v1_1_0:1_5", "/usr/bin/docker"),
		auditCheck("CIS_Docker_v1_1_0:1_6", "/var/lib/docker"),
		auditCheck("CIS_Docker_v1_1_0:1_7", "/etc/docker"),
		auditCheck("CIS_Docker_v1_1_0:1_8", "docker.service"),
		auditCheck("CIS_Docker_v1_1_0:1_9", "docker.socket"),
		auditCheck("CIS_Docker_v1_1_0:1_10", "/etc/default/docker"),
		auditCheck("CIS_Docker_v1_1_0:1_11", "/etc/docker/daemon.json"),
		auditCheck("CIS_Docker_v1_1_0:1_12", "/usr/bin/docker-containerd"),
		auditCheck("CIS_Docker_v1_1_0:1_13", "/usr/bin/docker-runc"),

		framework.NewCheckFromFunc("CIS_Docker_v1_1_0:5_22", framework.NodeKind, nil, privilegedDockerExec),
		framework.NewCheckFromFunc("CIS_Docker_v1_1_0:5_23", framework.NodeKind, nil, userDockerExec),
	)
}

const auditFile = "/etc/audit/audit.rules"

func auditCheck(name, file string) framework.Check {
	return framework.NewCheckFromFunc(name, framework.NodeKind, nil, auditCheckFunc(file))
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

const fstabPath = "/etc/fstab"

func containerPartition(funcCtx framework.ComplianceContext) {
	fileByte := []byte("/var/lib/docker")
	common.PerNodeCheck(func(ctx framework.ComplianceContext, returnData *compliance.ComplianceReturn) {
		f, ok := returnData.Files[fstabPath]
		if !ok {
			framework.FailNowf(ctx, "FStab file %q does not exist", fstabPath)
		}
		if !bytes.Contains(f.GetContent(), fileByte) {
			framework.FailNowf(ctx, "FStab file %q does not contain file /var/lib/docker", fstabPath)
		}
		framework.PassNowf(ctx, "FStab file %q contains file %q", fstabPath, "/var/lib/docker")
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
