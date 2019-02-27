package kubernetes

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

func init() {
	framework.MustRegisterChecks(
		common.OptionalPermissionCheck("CIS_Kubernetes_v1_2_0:1_4_1", "/etc/kubernetes/manifests/kube-apiserver.yaml", 0644),
		common.OptionalOwnershipCheck("CIS_Kubernetes_v1_2_0:1_4_2", "/etc/kubernetes/manifests/kube-apiserver.yaml", "root", "root"),

		common.OptionalPermissionCheck("CIS_Kubernetes_v1_2_0:1_4_3", "/etc/kubernetes/manifests/kube-controller-manager.yaml", 0644),
		common.OptionalOwnershipCheck("CIS_Kubernetes_v1_2_0:1_4_4", "/etc/kubernetes/manifests/kube-controller-manager.yaml", "root", "root"),

		common.OptionalPermissionCheck("CIS_Kubernetes_v1_2_0:1_4_5", "/etc/kubernetes/manifests/kube-controller-manager.yaml", 0644),
		common.OptionalOwnershipCheck("CIS_Kubernetes_v1_2_0:1_4_6", "/etc/kubernetes/manifests/kube-controller-manager.yaml", "root", "root"),

		common.OptionalPermissionCheck("CIS_Kubernetes_v1_2_0:1_4_7", "/etc/kubernetes/manifests/kube-scheduler.yaml", 0644),
		common.OptionalOwnershipCheck("CIS_Kubernetes_v1_2_0:1_4_8", "/etc/kubernetes/manifests/kube-scheduler.yaml", "root", "root"),

		cniFilePermissions(),
		cniFileOwnership(),

		etcdDataPermissions(),
		etcdDataOwnership(),

		common.OptionalPermissionCheck("CIS_Kubernetes_v1_2_0:1_4_13", "/etc/kubernetes/manifests/etcd.yaml", 0644),
		common.OptionalOwnershipCheck("CIS_Kubernetes_v1_2_0:1_4_14", "/etc/kubernetes/manifests/etcd.yaml", "root", "root"),

		common.OptionalPermissionCheck("CIS_Kubernetes_v1_2_0:1_4_15", "/etc/kubernetes/admin.conf", 0644),
		common.OptionalOwnershipCheck("CIS_Kubernetes_v1_2_0:1_4_16", "/etc/kubernetes/admin.conf", "root", "root"),

		common.OptionalPermissionCheck("CIS_Kubernetes_v1_2_0:1_4_17", "/etc/kubernetes/scheduler.conf", 0644),
		common.OptionalOwnershipCheck("CIS_Kubernetes_v1_2_0:1_4_18", "/etc/kubernetes/scheduler.conf", "root", "root"),
	)
}

func getDirectoryFileFromCommandLine(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn, processName string, flag, defaultVal string) *compliance.File {
	process, exists := common.GetProcess(ret, processName)
	if !exists {
		framework.NoteNowf(ctx, "Process %q not found on host therefore check is not applicable", processName)
	}
	var dir string
	values := common.GetValuesForCommandFromFlagsAndConfig(process.Args, nil, flag)
	if len(values) == 0 {
		dir = defaultVal
	} else {
		dir = values[0]
	}
	dirFile, exists := ret.Files[dir]
	if !exists {
		framework.Failf(ctx, "%q directory does not exist", dir)
		return nil
	}
	return dirFile
}

func cniFilePermissions() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Kubernetes_v1_2_0:1_4_9",
		Scope:              framework.NodeKind,
		InterpretationText: "StackRox checks that the permissions of files in the CNI configuration and binary directories are set to at most '0644'",
		DataDependencies:   []string{"HostScraped"},
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			if dirFile := getDirectoryFileFromCommandLine(ctx, ret, "kubelet", "cni-conf-dir", "/etc/cni/net.d"); dirFile != nil {
				common.CheckRecursivePermissions(ctx, dirFile, 0644)
			}
			if dirFile := getDirectoryFileFromCommandLine(ctx, ret, "kubelet", "cni-bin-dir", "/opt/cni/bin"); dirFile != nil {
				common.CheckRecursivePermissions(ctx, dirFile, 0644)
			}
		}))
}

func cniFileOwnership() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Kubernetes_v1_2_0:1_4_10",
		Scope:              framework.NodeKind,
		InterpretationText: "StackRox checks that the owner and group of files in the CNI configuration and binary directories is root:root",
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			if dirFile := getDirectoryFileFromCommandLine(ctx, ret, "kubelet", "cni-conf-dir", "/etc/cni/net.d"); dirFile != nil {
				common.CheckRecursiveOwnership(ctx, dirFile, "root", "root")
			}
			if dirFile := getDirectoryFileFromCommandLine(ctx, ret, "kubelet", "cni-bin-dir", "/opt/cni/bin"); dirFile != nil {
				common.CheckRecursiveOwnership(ctx, dirFile, "root", "root")
			}
		}))
}

func etcdDataPermissions() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Kubernetes_v1_2_0:1_4_11",
		Scope:              framework.NodeKind,
		InterpretationText: "StackRox checks that the permissions of the etcd data directory are set to '0700'",
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			if dirFile := getDirectoryFileFromCommandLine(ctx, ret, "etcd", "data-dir", "/var/lib/etcddisk"); dirFile != nil {
				common.CheckRecursivePermissions(ctx, dirFile, 0700)
			}
		}))
}

func etcdDataOwnership() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Kubernetes_v1_2_0:1_4_12",
		Scope:              framework.NodeKind,
		InterpretationText: "StackRox checks that the owner and group of the etcd data directory are set to etcd:etcd",
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			if dirFile := getDirectoryFileFromCommandLine(ctx, ret, "etcd", "data-dir", "/var/lib/etcddisk"); dirFile != nil {
				common.CheckRecursiveOwnership(ctx, dirFile, "etcd", "etcd")
			}
		}))
}
